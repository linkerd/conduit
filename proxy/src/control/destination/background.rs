use futures::sync::mpsc;
use futures::{Async, Future, Stream};
use std::collections::{
    hash_map::{Entry, HashMap},
    VecDeque,
};
use std::fmt;
use std::iter::IntoIterator;
use std::net::SocketAddr;
use std::time::Duration;
use tower_grpc as grpc;
use tower_h2::{BoxBody, HttpService, RecvBody};

use conduit_proxy_controller_grpc::common::{Destination, TcpAddress};
use conduit_proxy_controller_grpc::destination::client::Destination as DestinationSvc;
use conduit_proxy_controller_grpc::destination::update::Update as PbUpdate2;
use conduit_proxy_controller_grpc::destination::{Update as PbUpdate, WeightedAddr};

use super::{Metadata, ResolveRequest, Responder, Update};
use control::cache::{Cache, CacheChange, Exists};
use control::fully_qualified_authority::FullyQualifiedAuthority;
use control::remote_stream::{Receiver, Remote};
use dns::{self, IpAddrListFuture};
use telemetry::metrics::DstLabels;
use transport::DnsNameAndPort;

type DestinationServiceQuery<T> = Remote<PbUpdate, T>;
type UpdateRx<T> = Receiver<PbUpdate, T>;

/// Stores the configuration for a destination background worker.
#[derive(Debug)]
pub struct Config {
    request_rx: mpsc::UnboundedReceiver<ResolveRequest>,
    dns_config: dns::Config,
    default_destination_namespace: String,
}

/// Satisfies resolutions as requested via `request_rx`.
///
/// As `Process` is polled with a client to Destination service, if the client to the
/// service is healthy, it reads requests from `request_rx`, determines how to resolve the
/// provided authority to a set of addresses, and ensures that resolution updates are
/// propagated to all requesters.
pub struct Process<T: HttpService<ResponseBody = RecvBody>> {
    dns_resolver: dns::Resolver,
    default_destination_namespace: String,
    destinations: HashMap<DnsNameAndPort, DestinationSet<T>>,
    /// A queue of authorities that need to be reconnected.
    reconnects: VecDeque<DnsNameAndPort>,
    /// The Destination.Get RPC client service.
    /// Each poll, records whether the rpc service was till ready.
    rpc_ready: bool,
    /// A receiver of new watch requests.
    request_rx: mpsc::UnboundedReceiver<ResolveRequest>,
}

/// Holds the state of a single resolution.
struct DestinationSet<T: HttpService<ResponseBody = RecvBody>> {
    addrs: Exists<Cache<SocketAddr, Metadata>>,
    query: Option<DestinationServiceQuery<T>>,
    dns_query: Option<IpAddrListFuture>,
    responders: Vec<Responder>,
}

// ==== impl Config =====

impl Config {
    pub(super) fn new(
        request_rx: mpsc::UnboundedReceiver<ResolveRequest>,
        dns_config: dns::Config,
        default_destination_namespace: String,
    ) -> Self {
        Self {
            request_rx,
            dns_config,
            default_destination_namespace,
        }
    }

    /// Bind this handle to start talking to the controller API.
    pub fn process<T>(self) -> Process<T>
    where
        T: HttpService<RequestBody = BoxBody, ResponseBody = RecvBody>,
        T::Error: fmt::Debug,
    {
        Process {
            dns_resolver: dns::Resolver::new(self.dns_config),
            default_destination_namespace: self.default_destination_namespace,
            destinations: HashMap::new(),
            reconnects: VecDeque::new(),
            rpc_ready: false,
            request_rx: self.request_rx,
        }
    }
}

// ==== impl Process =====

impl<T> Process<T>
where
    T: HttpService<RequestBody = BoxBody, ResponseBody = RecvBody>,
    T::Error: fmt::Debug,
{
    pub fn poll_rpc(&mut self, client: &mut T) {
        // This loop is make sure any streams that were found disconnected
        // in `poll_destinations` while the `rpc` service is ready should
        // be reconnected now, otherwise the task would just sleep...
        loop {
            self.poll_resolve_requests(client);
            self.retain_active_destinations();
            self.poll_destinations();

            if self.reconnects.is_empty() || !self.rpc_ready {
                break;
            }
        }
    }

    fn poll_resolve_requests(&mut self, client: &mut T) {
        loop {
            // if rpc service isn't ready, not much we can do...
            match client.poll_ready() {
                Ok(Async::Ready(())) => {
                    self.rpc_ready = true;
                },
                Ok(Async::NotReady) => {
                    self.rpc_ready = false;
                    break;
                },
                Err(err) => {
                    warn!("Destination.Get poll_ready error: {:?}", err);
                    self.rpc_ready = false;
                    break;
                },
            }

            // handle any pending reconnects first
            if self.poll_reconnect(client) {
                continue;
            }

            // check for any new watches
            match self.request_rx.poll() {
                Ok(Async::Ready(Some(resolve))) => {
                    trace!("Destination.Get {:?}", resolve.authority);
                    match self.destinations.entry(resolve.authority) {
                        Entry::Occupied(mut occ) => {
                            let set = occ.get_mut();
                            // we may already know of some addresses here, so push
                            // them onto the new watch first
                            match set.addrs {
                                Exists::Yes(ref cache) => for (&addr, meta) in cache {
                                    let update = Update::Insert(addr, meta.clone());
                                    resolve.responder.update_tx
                                        .unbounded_send(update)
                                        .expect("unbounded_send does not fail");
                                },
                                Exists::No | Exists::Unknown => (),
                            }
                            set.responders.push(resolve.responder);
                        },
                        Entry::Vacant(vac) => {
                            let query = Self::query_destination_service_if_relevant(
                                &self.default_destination_namespace,
                                client,
                                vac.key(),
                                "connect",
                            );
                            let mut set = DestinationSet {
                                addrs: Exists::Unknown,
                                query,
                                dns_query: None,
                                responders: vec![resolve.responder],
                            };
                            // If the authority is one for which the Destination service is never
                            // relevant (e.g. an absolute name that doesn't end in ".svc.$zone." in
                            // Kubernetes), then immediately start polling DNS.
                            if set.query.is_none() {
                                set.reset_dns_query(
                                    &self.dns_resolver,
                                    Duration::from_secs(0),
                                    vac.key(),
                                );
                            }
                            vac.insert(set);
                        },
                    }
                },
                Ok(Async::Ready(None)) => {
                    trace!("Discover tx is dropped, shutdown?");
                    return;
                },
                Ok(Async::NotReady) => break,
                Err(_) => unreachable!("unbounded receiver doesn't error"),
            }
        }
    }

    /// Tries to reconnect next watch stream. Returns true if reconnection started.
    fn poll_reconnect(&mut self, client: &mut T) -> bool {
        debug_assert!(self.rpc_ready);

        while let Some(auth) = self.reconnects.pop_front() {
            if let Some(set) = self.destinations.get_mut(&auth) {
                set.query = Self::query_destination_service_if_relevant(
                    &self.default_destination_namespace,
                    client,
                    &auth,
                    "reconnect",
                );
                return true;
            } else {
                trace!("reconnect no longer needed: {:?}", auth);
            }
        }
        false
    }

    /// Ensures that `destinations` is updated to only maintain active resolutions.
    ///
    /// If there are no active resolutions for a destination, the destination is removed.
    fn retain_active_destinations(&mut self) {
        self.destinations.retain(|_, ref mut dst| {
            dst.responders.retain(|r| r.is_active());
            dst.responders.len() > 0
        })
    }

    fn poll_destinations(&mut self) {
        for (auth, set) in &mut self.destinations {
            // Query the Destination service first.
            let (new_query, found_by_destination_service) = match set.query.take() {
                Some(Remote::ConnectedOrConnecting { rx }) => {
                    let (new_query, found_by_destination_service) =
                        set.poll_destination_service(auth, rx);
                    if let Remote::NeedsReconnect = new_query {
                        set.reset_on_next_modification();
                        self.reconnects.push_back(auth.clone());
                    }
                    (Some(new_query), found_by_destination_service)
                },
                query => (query, Exists::Unknown),
            };
            set.query = new_query;

            // Any active response from the Destination service cancels the DNS query except for a
            // positive assertion that the service doesn't exist.
            //
            // Any disconnection from the Destination service has no effect on the DNS query; we
            // assume that if we were querying DNS before, we should continue to do so, and if we
            // weren't querying DNS then we shouldn't start now. In particular, temporary
            // disruptions of connectivity to the Destination service do not cause a fallback to
            // DNS.
            match found_by_destination_service {
                Exists::Yes(()) => {
                    // Stop polling DNS on any active update from the Destination service.
                    set.dns_query = None;
                },
                Exists::No => {
                    // Fall back to DNS.
                    set.reset_dns_query(&self.dns_resolver, Duration::from_secs(0), auth);
                },
                Exists::Unknown => (), // No change from Destination service's perspective.
            }

            // Poll DNS after polling the Destination service. This may reset the DNS query but it
            // won't affect the Destination Service query.
            set.poll_dns(&self.dns_resolver, auth);
        }
    }

    /// Initiates a query `query` to the Destination service and returns it as
    /// `Some(query)` if the given authority's host is of a form suitable for using to
    /// query the Destination service. Otherwise, returns `None`.
    fn query_destination_service_if_relevant(
        default_destination_namespace: &str,
        client: &mut T,
        auth: &DnsNameAndPort,
        connect_or_reconnect: &str,
    ) -> Option<DestinationServiceQuery<T>> {
        trace!(
            "DestinationServiceQuery {} {:?}",
            connect_or_reconnect,
            auth
        );
        FullyQualifiedAuthority::normalize(auth, default_destination_namespace).map(|auth| {
            let req = Destination {
                scheme: "k8s".into(),
                path: auth.without_trailing_dot().to_owned(),
            };
            let mut svc = DestinationSvc::new(client.lift_ref());
            let response = svc.get(grpc::Request::new(req));
            Remote::ConnectedOrConnecting {
                rx: Receiver::new(response),
            }
        })
    }
}

// ===== impl DestinationSet =====

impl<T> DestinationSet<T>
where
    T: HttpService<RequestBody = BoxBody, ResponseBody = RecvBody>,
    T::Error: fmt::Debug,
{
    fn reset_dns_query(
        &mut self,
        dns_resolver: &dns::Resolver,
        delay: Duration,
        authority: &DnsNameAndPort,
    ) {
        trace!(
            "resetting DNS query for {} with delay {:?}",
            authority.host,
            delay
        );
        self.reset_on_next_modification();
        self.dns_query = Some(dns_resolver.resolve_all_ips(delay, &authority.host));
    }

    // Processes Destination service updates from `request_rx`, returning the new query
    // and an indication of any *change* to whether the service exists as far as the
    // Destination service is concerned, where `Exists::Unknown` is to be interpreted as
    // "no change in existence" instead of "unknown".
    fn poll_destination_service(
        &mut self,
        auth: &DnsNameAndPort,
        mut rx: UpdateRx<T>,
    ) -> (DestinationServiceQuery<T>, Exists<()>) {
        let mut exists = Exists::Unknown;

        loop {
            match rx.poll() {
                Ok(Async::Ready(Some(update))) => match update.update {
                    Some(PbUpdate2::Add(a_set)) => {
                        let set_labels = a_set.metric_labels;
                        let addrs = a_set
                            .addrs
                            .into_iter()
                            .filter_map(|pb| pb_to_addr_meta(pb, &set_labels));
                        self.add(auth, addrs)
                    },
                    Some(PbUpdate2::Remove(r_set)) => {
                        exists = Exists::Yes(());
                        self.remove(
                            auth,
                            r_set
                                .addrs
                                .iter()
                                .filter_map(|addr| pb_to_sock_addr(addr.clone())),
                        );
                    },
                    Some(PbUpdate2::NoEndpoints(ref no_endpoints)) if no_endpoints.exists => {
                        exists = Exists::Yes(());
                        self.no_endpoints(auth, no_endpoints.exists);
                    },
                    Some(PbUpdate2::NoEndpoints(no_endpoints)) => {
                        debug_assert!(!no_endpoints.exists);
                        exists = Exists::No;
                    },
                    None => (),
                },
                Ok(Async::Ready(None)) => {
                    trace!(
                        "Destination.Get stream ended for {:?}, must reconnect",
                        auth
                    );
                    return (Remote::NeedsReconnect, exists);
                },
                Ok(Async::NotReady) => {
                    return (Remote::ConnectedOrConnecting { rx }, exists);
                },
                Err(err) => {
                    warn!("Destination.Get stream errored for {:?}: {:?}", auth, err);
                    return (Remote::NeedsReconnect, exists);
                },
            };
        }
    }

    fn poll_dns(&mut self, dns_resolver: &dns::Resolver, authority: &DnsNameAndPort) {
        trace!("checking DNS for {:?}", authority);
        while let Some(mut query) = self.dns_query.take() {
            trace!("polling DNS for {:?}", authority);
            match query.poll() {
                Ok(Async::NotReady) => {
                    trace!("DNS query not ready {:?}", authority);
                    self.dns_query = Some(query);
                    return;
                },
                Ok(Async::Ready(dns::Response::Exists(ips))) => {
                    trace!(
                        "positive result of DNS query for {:?}: {:?}",
                        authority,
                        ips
                    );
                    self.add(
                        authority,
                        ips.iter().map(|ip| {
                            (
                                SocketAddr::from((ip, authority.port)),
                                Metadata::no_metadata(),
                            )
                        }),
                    );
                },
                Ok(Async::Ready(dns::Response::DoesNotExist)) => {
                    trace!(
                        "negative result (NXDOMAIN) of DNS query for {:?}",
                        authority
                    );
                    self.no_endpoints(authority, false);
                },
                Err(e) => {
                    trace!("DNS resolution failed for {}: {}", &authority.host, e);
                    // Do nothing so that the most recent non-error response is used until a
                    // non-error response is received.
                },
            };
            // TODO: When we have a TTL to use, we should use that TTL instead of hard-coding this
            // delay.
            self.reset_dns_query(dns_resolver, Duration::from_secs(5), &authority)
        }
    }
}

impl<T: HttpService<ResponseBody = RecvBody>> DestinationSet<T> {
    fn reset_on_next_modification(&mut self) {
        match self.addrs {
            Exists::Yes(ref mut cache) => {
                cache.set_reset_on_next_modification();
            },
            Exists::No | Exists::Unknown => (),
        }
    }

    fn add<A>(&mut self, authority_for_logging: &DnsNameAndPort, addrs_to_add: A)
    where
        A: Iterator<Item = (SocketAddr, Metadata)>,
    {
        let mut cache = match self.addrs.take() {
            Exists::Yes(mut cache) => cache,
            Exists::Unknown | Exists::No => Cache::new(),
        };
        cache.update_union(addrs_to_add, &mut |change| {
            Self::on_change(&mut self.responders, authority_for_logging, change)
        });
        self.addrs = Exists::Yes(cache);
    }

    fn remove<A>(&mut self, authority_for_logging: &DnsNameAndPort, addrs_to_remove: A)
    where
        A: Iterator<Item = SocketAddr>,
    {
        let cache = match self.addrs.take() {
            Exists::Yes(mut cache) => {
                cache.remove(addrs_to_remove, &mut |change| {
                    Self::on_change(&mut self.responders, authority_for_logging, change)
                });
                cache
            },
            Exists::Unknown | Exists::No => Cache::new(),
        };
        self.addrs = Exists::Yes(cache);
    }

    fn no_endpoints(&mut self, authority_for_logging: &DnsNameAndPort, exists: bool) {
        trace!(
            "no endpoints for {:?} that is known to {}",
            authority_for_logging,
            if exists { "exist" } else { "not exist" }
        );
        match self.addrs.take() {
            Exists::Yes(mut cache) => {
                cache.clear(&mut |change| {
                    Self::on_change(&mut self.responders, authority_for_logging, change)
                });
            },
            Exists::Unknown | Exists::No => (),
        };
        self.addrs = if exists {
            Exists::Yes(Cache::new())
        } else {
            Exists::No
        };
    }

    fn on_change(
        responders: &mut Vec<Responder>,
        authority_for_logging: &DnsNameAndPort,
        change: CacheChange<SocketAddr, Metadata>,
    ) {
        let (update_str, update, addr) = match change {
            CacheChange::Insertion { key, value } => {
                ("insert", Update::Insert(key, value.clone()), key)
            },
            CacheChange::Removal { key } => ("remove", Update::Remove(key), key),
            CacheChange::Modification { key, new_value } => (
                "change metadata for",
                Update::ChangeMetadata(key, new_value.clone()),
                key,
            ),
        };
        trace!("{} {:?} for {:?}", update_str, addr, authority_for_logging);
        // retain is used to drop any senders that are dead
        responders.retain(|r| {
            let sent = r.update_tx.unbounded_send(update.clone());
            sent.is_ok()
        });
    }
}

/// Construct a new labeled `SocketAddr `from a protobuf `WeightedAddr`.
fn pb_to_addr_meta(
    pb: WeightedAddr,
    set_labels: &HashMap<String, String>,
) -> Option<(SocketAddr, Metadata)> {
    let addr = pb.addr.and_then(pb_to_sock_addr)?;
    let label_iter = set_labels.iter().chain(pb.metric_labels.iter());
    let meta = Metadata {
        metric_labels: DstLabels::new(label_iter),
    };
    Some((addr, meta))
}

fn pb_to_sock_addr(pb: TcpAddress) -> Option<SocketAddr> {
    use conduit_proxy_controller_grpc::common::ip_address::Ip;
    use std::net::{Ipv4Addr, Ipv6Addr};
    /*
    current structure is:
    TcpAddress {
        ip: Option<IpAddress {
            ip: Option<enum Ip {
                Ipv4(u32),
                Ipv6(IPv6 {
                    first: u64,
                    last: u64,
                }),
            }>,
        }>,
        port: u32,
    }
    */
    match pb.ip {
        Some(ip) => match ip.ip {
            Some(Ip::Ipv4(octets)) => {
                let ipv4 = Ipv4Addr::from(octets);
                Some(SocketAddr::from((ipv4, pb.port as u16)))
            },
            Some(Ip::Ipv6(v6)) => {
                let octets = [
                    (v6.first >> 56) as u8,
                    (v6.first >> 48) as u8,
                    (v6.first >> 40) as u8,
                    (v6.first >> 32) as u8,
                    (v6.first >> 24) as u8,
                    (v6.first >> 16) as u8,
                    (v6.first >> 8) as u8,
                    v6.first as u8,
                    (v6.last >> 56) as u8,
                    (v6.last >> 48) as u8,
                    (v6.last >> 40) as u8,
                    (v6.last >> 32) as u8,
                    (v6.last >> 24) as u8,
                    (v6.last >> 16) as u8,
                    (v6.last >> 8) as u8,
                    v6.last as u8,
                ];
                let ipv6 = Ipv6Addr::from(octets);
                Some(SocketAddr::from((ipv6, pb.port as u16)))
            },
            None => None,
        },
        None => None,
    }
}
