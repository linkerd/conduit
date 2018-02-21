use std::net::SocketAddr;

use futures::{Async, Poll};
use http;
use rand;
use std::sync::Arc;
use tower;
use tower_balance::{self, choose, load, Balance};
use tower_buffer::Buffer;
use tower_discover::{Change, Discover};
use tower_h2;
use conduit_proxy_router::Recognize;

use bind::{self, Bind, Protocol};
use control::{self, discovery};
use control::discovery::Bind as BindTrait;
use ctx;
use fully_qualified_authority::FullyQualifiedAuthority;

type BindProtocol<B> = bind::BindProtocol<Arc<ctx::Proxy>, B>;

pub struct Outbound<B> {
    bind: Bind<Arc<ctx::Proxy>, B>,
    discovery: control::Control,
    default_namespace: Option<String>,
    default_zone: Option<String>,
}

// ===== impl Outbound =====

impl<B> Outbound<B> {
    pub fn new(bind: Bind<Arc<ctx::Proxy>, B>, discovery: control::Control,
               default_namespace: Option<String>, default_zone: Option<String>)
               -> Outbound<B> {
        Self {
            bind,
            discovery,
            default_namespace,
            default_zone,
        }
    }
}

#[derive(Clone, Debug, PartialEq, Eq, Hash)]
pub enum Destination {
    LocalSvc(FullyQualifiedAuthority),
    External(SocketAddr),
}

impl<B> Recognize for Outbound<B>
where
    B: tower_h2::Body + 'static,
{
    type Request = http::Request<B>;
    type Response = bind::HttpResponse;
    type Error = <Self::Service as tower::Service>::Error;
    type Key = (Destination, Protocol);
    type RouteError = ();
    type Service = Buffer<Balance<
        load::WithPendingRequests<Discovery<B>>,
        choose::PowerOfTwoChoices<rand::ThreadRng>,
    >>;

    fn recognize(&self, req: &Self::Request) -> Option<Self::Key> {
        let local = req.uri().authority_part().and_then(|authority| {
            FullyQualifiedAuthority::normalize(
                authority,
                self.default_namespace.as_ref().map(|s| s.as_ref()),
                self.default_zone.as_ref().map(|s| s.as_ref()))

        });

        // If we can't fully qualify the authority as a local service,
        // and there is no original dst, then we have nothing! In that
        // case, we return `None`, which results an "unrecognized" error.
        //
        // In practice, this shouldn't ever happen, since we expect the proxy
        // to be run on Linux servers, with iptables setup, so there should
        // always be an original destination.
        let dest = if let Some(local) = local {
            Destination::LocalSvc(local)
        } else {
            let orig_dst = req.extensions()
                .get::<Arc<ctx::transport::Server>>()
                .and_then(|ctx| {
                    ctx.orig_dst_if_not_local()
                });
            Destination::External(orig_dst?)
        };

        let proto = match req.version() {
            http::Version::HTTP_2 => Protocol::Http2,
            _ => Protocol::Http1,
        };

        Some((dest, proto))
    }

    /// Builds a dynamic, load balancing service.
    ///
    /// Resolves the authority in service discovery and initializes a service that buffers
    /// and load balances requests across.
    ///
    /// # TODO
    ///
    /// Buffering is currently unbounded and does not apply timeouts. This must be
    /// changed.
    fn bind_service(
        &mut self,
        key: &Self::Key,
    ) -> Result<Self::Service, Self::RouteError> {
        let &(ref dest, protocol) = key;
        debug!("building outbound {:?} client to {:?}", protocol, dest);

        let resolve = match *dest {
            Destination::LocalSvc(ref authority) => {
                Discovery::LocalSvc(self.discovery.resolve(
                    authority,
                    self.bind.clone().with_protocol(protocol),
                ))
            },
            Destination::External(addr) => {
                Discovery::External(Some((addr, self.bind.clone().with_protocol(protocol))))
            }
        };

        let loaded = tower_balance::load::WithPendingRequests::new(resolve);

        let balance = tower_balance::power_of_two_choices(loaded, rand::thread_rng());

        // Wrap with buffering. This currently is an unbounded buffer,
        // which is not ideal.
        //
        // TODO: Don't use unbounded buffering.
        Buffer::new(balance, self.bind.executor()).map_err(|_| {})
    }
}

pub enum Discovery<B> {
    LocalSvc(discovery::Watch<BindProtocol<B>>),
    External(Option<(SocketAddr, BindProtocol<B>)>),
}

impl<B> Discover for Discovery<B>
where
    B: tower_h2::Body + 'static,
{
    type Key = SocketAddr;
    type Request = http::Request<B>;
    type Response = bind::HttpResponse;
    type Error = <bind::Service<B> as tower::Service>::Error;
    type Service = bind::Service<B>;
    type DiscoverError = ();

    fn poll(&mut self) -> Poll<Change<Self::Key, Self::Service>, Self::DiscoverError> {
        match *self {
            Discovery::LocalSvc(ref mut w) => w.poll(),
            Discovery::External(ref mut opt) => {
                // This "discovers" a single address for an external service
                // that never has another change. This can mean it floats
                // in the Balancer forever. However, when we finally add
                // circuit-breaking, this should be able to take care of itself,
                // closing down when the connection is no longer usable.
                if let Some((addr, bind)) = opt.take() {
                    let svc = bind.bind(&addr)?;
                    Ok(Async::Ready(Change::Insert(addr, svc)))
                } else {
                    Ok(Async::NotReady)
                }
            }
        }
    }
}
