use std::{
    fs::File,
    io::{self, Cursor, Read},
    path::PathBuf,
    sync::Arc,
    time::Duration,
};

use super::{
    cert_resolver::CertResolver,
    Identity,

    rustls,
    untrusted,
    webpki,
};

use futures::{future, stream, Future, Stream};
use futures_watch::Watch;

/// Not-yet-validated settings that are used for both TLS clients and TLS
/// servers.
///
/// The trust anchors are stored in PEM format because, in Kubernetes, they are
/// stored in a ConfigMap, and until very recently Kubernetes cannot store
/// binary data in ConfigMaps. Also, PEM is the most interoperable way to
/// distribute trust anchors, especially if it is desired to support multiple
/// trust anchors at once.
///
/// The end-entity certificate and private key are in DER format because they
/// are stored in the secret store where space utilization is a concern, and
/// because PEM doesn't offer any advantages.
#[derive(Clone, Debug)]
pub struct CommonSettings {
    /// The trust anchors as concatenated PEM-encoded X.509 certificates.
    pub trust_anchors: PathBuf,

    /// The end-entity certificate as a DER-encoded X.509 certificate.
    pub end_entity_cert: PathBuf,

    /// The private key in DER-encoded PKCS#8 form.
    pub private_key: PathBuf,

    /// The identity we use to identify the service being proxied (as opposed
    /// to the psuedo-service exposed on the proxy's control port).
    pub service_identity: Identity,
}

/// Validated configuration common between TLS clients and TLS servers.
#[derive(Debug)]
struct CommonConfig {
    root_cert_store: rustls::RootCertStore,
    cert_resolver: Arc<CertResolver>,
}

/// Validated configuration for TLS clients.
///
/// TODO: Fill this in with the actual configuration.
#[derive(Clone, Debug)]
pub struct ClientConfig(Arc<()>);

/// Validated configuration for TLS servers.
#[derive(Clone)]
pub struct ServerConfig(pub(super) Arc<rustls::ServerConfig>);

pub type ClientConfigWatch = Watch<Option<ClientConfig>>;
pub type ServerConfigWatch = Watch<Option<ServerConfig>>;

#[derive(Debug)]
pub enum Error {
    Io(PathBuf, io::Error),
    FailedToParseTrustAnchors(Option<webpki::Error>),
    EndEntityCertIsNotValid(rustls::TLSError),
    InvalidPrivateKey,
}

impl CommonSettings {
    fn paths(&self) -> [&PathBuf; 3] {
        [
            &self.trust_anchors,
            &self.end_entity_cert,
            &self.private_key,
        ]
    }

    /// Stream changes to the files described by this `CommonSettings`.
    ///
    /// The returned stream consists of each subsequent successfully loaded
    /// `CommonSettings` after each change. If the settings could not be
    /// reloaded (i.e., they were malformed), nothing is sent.
    fn stream_changes(self, interval: Duration)
        -> impl Stream<Item = CommonConfig, Error = ()>
    {
        let paths = self.paths().iter()
            .map(|&p| p.clone())
            .collect::<Vec<_>>();
        // Generate one "change" immediately before starting to watch
        // the files, so that we'll try to load them now if they exist.
        stream::once(Ok(()))
            .chain(::fs_watch::stream_changes(paths, interval))
            .filter_map(move |_| {
                CommonConfig::load_from_disk(&self)
                    .map_err(|e| warn!("error reloading TLS config: {:?}, falling back", e))
                    .ok()
            })
    }

}

impl CommonConfig {
    /// Loads a configuration from the given files and validates it. If an
    /// error is returned then the caller should try again after the files are
    /// updated.
    ///
    /// In a valid configuration, all the files need to be in sync with each
    /// other. For example, the private key file must contain the private
    /// key for the end-entity certificate, and the end-entity certificate
    /// must be issued by the CA represented by a certificate in the
    /// trust anchors file. Since filesystem operations are not atomic, we
    /// need to check for this consistency.
    fn load_from_disk(settings: &CommonSettings) -> Result<Self, Error> {
        let root_cert_store = load_file_contents(&settings.trust_anchors)
            .and_then(|file_contents| {
                let mut root_cert_store = rustls::RootCertStore::empty();
                let (added, skipped) = root_cert_store.add_pem_file(&mut Cursor::new(file_contents))
                    .map_err(|err| {
                        error!("error parsing trust anchors file: {:?}", err);
                        Error::FailedToParseTrustAnchors(None)
                    })?;
                if skipped != 0 {
                    warn!("skipped {} trust anchors in trust anchors file", skipped);
                }
                if added > 0 {
                    Ok(root_cert_store)
                } else {
                    error!("no valid trust anchors in trust anchors file");
                    Err(Error::FailedToParseTrustAnchors(None))
                }
            })?;

        let end_entity_cert = load_file_contents(&settings.end_entity_cert)?;

        // XXX: Assume there are no intermediates since there is no way to load
        // them yet.
        let cert_chain = vec![rustls::Certificate(end_entity_cert)];

        // Load the private key after we've validated the certificate.
        let private_key = load_file_contents(&settings.private_key)?;
        let private_key = untrusted::Input::from(&private_key);

        // Ensure the certificate is valid for the services we terminate for
        // TLS. This assumes that server cert validation does the same or
        // more validation than client cert validation.
        //
        // XXX: Rustls currently only provides access to a
        // `ServerCertVerifier` through
        // `rustls::ClientConfig::get_verifier()`.
        //
        // XXX: Once `rustls::ServerCertVerified` is exposed in Rustls's
        // safe API, remove the `map(|_| ())` below.
        //
        // TODO: Restrict accepted signatutre algorithms.
        let certificate_was_validated =
            rustls::ClientConfig::new().get_verifier().verify_server_cert(
                    &root_cert_store,
                    &cert_chain,
                    settings.service_identity.as_dns_name_ref(),
                    &[]) // No OCSP
                .map(|_| ())
                .map_err(Error::EndEntityCertIsNotValid)?;

        // `CertResolver::new` is responsible for verifying that the
        // private key is the right one for the certificate.
        let cert_resolver = CertResolver::new(certificate_was_validated, cert_chain, private_key)?;

        Ok(Self {
            root_cert_store,
            cert_resolver: Arc::new(cert_resolver),
        })
    }

}

pub fn watch_for_config_changes(settings: Option<&CommonSettings>)
    -> (ClientConfigWatch, ServerConfigWatch, Box<Future<Item = (), Error = ()> + Send>)
{
    let settings = if let Some(settings) = settings {
        settings.clone()
    } else {
        let (client_watch, _) = Watch::new(None);
        let (server_watch, _) = Watch::new(None);
        let no_future = future::ok(());
        return (client_watch, server_watch, Box::new(no_future));
    };

    let changes = settings.stream_changes(Duration::from_secs(1));
    let (client_watch, client_store) = Watch::new(None);
    let (server_watch, server_store) = Watch::new(None);

    // `Store::store` will return an error iff all watchers have been dropped,
    // so we'll use `fold` to cancel the forwarding future. Eventually, we can
    // also use the fold to continue tracking previous states if we need to do
    // that.
    let f = changes
        .fold(
            (client_store, server_store),
            |(mut client_store, mut server_store), ref config| {
                client_store
                    .store(Some(ClientConfig(Arc::new(()))))
                    .map_err(|_| trace!("all client config watchers dropped"))?;
                server_store
                    .store(Some(ServerConfig::from(config)))
                    .map_err(|_| trace!("all server config watchers dropped"))?;
                Ok((client_store, server_store))
            })
        .then(|_| {
            trace!("forwarding to server config watch finished.");
            Ok(())
        });

    // This function and `ServerConfig::no_tls` return `Box<Future<...>>`
    // rather than `impl Future<...>` so that they can have the _same_ return
    // types (impl Traits are not the same type unless the original
    // non-anonymized type was the same).
    (client_watch, server_watch, Box::new(f))
}

impl ServerConfig {
    fn from(common: &CommonConfig) -> Self {
        // Ask TLS clients for a certificate and accept any certificate issued
        // by our trusted CA(s).
        //
        // Initially, also allow TLS clients that don't have a client
        // certificate too, to minimize risk. TODO: Use
        // `AllowAnyAuthenticatedClient` to require a valid certificate.
        //
        // XXX: Rustls's built-in verifiers don't let us tweak things as fully
        // as we'd like (e.g. controlling the set of trusted signature
        // algorithms), but they provide good enough defaults for now.
        // TODO: lock down the verification further.
        //
        // TODO: Change Rustls's API to Avoid needing to clone `root_cert_store`.
        let client_cert_verifier =
            rustls::AllowAnyAnonymousOrAuthenticatedClient::new(common.root_cert_store.clone());

        let mut config = rustls::ServerConfig::new(client_cert_verifier);
        set_common_settings(&mut config.versions);
        config.cert_resolver = common.cert_resolver.clone();
        ServerConfig(Arc::new(config))
    }
}

fn load_file_contents(path: &PathBuf) -> Result<Vec<u8>, Error> {
    fn load_file(path: &PathBuf) -> Result<Vec<u8>, io::Error> {
        let mut result = Vec::new();
        let mut file = File::open(path)?;
        loop {
            match file.read_to_end(&mut result) {
                Ok(_) => {
                    return Ok(result);
                },
                Err(e) => {
                    if e.kind() != io::ErrorKind::Interrupted {
                        return Err(e);
                    }
                },
            }
        }
    }

    load_file(path)
        .map(|contents| {
            trace!("loaded file {:?}", path);
            contents
        })
        .map_err(|e| Error::Io(path.clone(), e))
}

fn set_common_settings(versions: &mut Vec<rustls::ProtocolVersion>) {
    // Only enable TLS 1.2 until TLS 1.3 is stable.
    *versions = vec![rustls::ProtocolVersion::TLSv1_2]
}


#[cfg(test)]
mod tests {
    use tls::{CommonSettings, Identity, ServerConfig};
    use super::{CommonConfig, Error};
    use config::Namespaces;
    use std::path::PathBuf;

    struct Strings {
        pod_name: &'static str,
        pod_ns: &'static str,
        controller_ns: &'static str,
        trust_anchors: &'static str,
        end_entity_cert: &'static str,
        private_key: &'static str,
    }

    fn settings(s: &Strings) -> CommonSettings {
        let dir = PathBuf::from("src/transport/tls/testdata");
        let namespaces = Namespaces {
            pod: s.pod_ns.into(),
            tls_controller: Some(s.controller_ns.into()),
        };
        let service_identity = Identity::try_from_pod_name(&namespaces, s.pod_name).unwrap();
        CommonSettings {
            trust_anchors: dir.join(s.trust_anchors),
            end_entity_cert: dir.join(s.end_entity_cert),
            private_key: dir.join(s.private_key),
            service_identity,
        }
    }

    #[test]
    fn can_construct_server_config_from_valid_settings() {
        let settings = settings(&Strings {
            pod_name: "foo",
            pod_ns: "ns1",
            controller_ns: "conduit",
            trust_anchors: "ca1.pem",
            end_entity_cert: "foo-ns1-ca1.crt",
            private_key: "foo-ns1-ca1.p8",
        });
        let config = CommonConfig::load_from_disk(&settings).unwrap();
        let _: ServerConfig = ServerConfig::from(&config); // Infallible.
    }

    #[test]
    fn recognize_ca_did_not_issue_cert() {
        let settings = settings(&Strings {
            pod_name: "foo",
            pod_ns: "ns1",
            controller_ns: "conduit",
            trust_anchors: "ca2.pem", // Mismatch
            end_entity_cert: "foo-ns1-ca1.crt",
            private_key: "foo-ns1-ca1.p8",
        });
        match CommonConfig::load_from_disk(&settings) {
            Err(Error::EndEntityCertIsNotValid(_)) => (),
            r => unreachable!("CommonConfig::load_from_disk returned {:?}", r),
        }
    }

    #[test]
    fn recognize_cert_is_not_valid_for_identity() {
        let settings = settings(&Strings {
            pod_name: "foo", // Mismatch
            pod_ns: "ns1",
            controller_ns: "conduit",
            trust_anchors: "ca1.pem",
            end_entity_cert: "bar-ns1-ca1.crt",
            private_key: "bar-ns1-ca1.p8",
        });
        match CommonConfig::load_from_disk(&settings) {
            Err(Error::EndEntityCertIsNotValid(_)) => (),
            r => unreachable!("CommonConfig::load_from_disk returned {:?}", r),
        }
    }

    // XXX: The check that this tests hasn't been implemented yet.
    #[test]
    #[should_panic]
    fn recognize_private_key_is_not_valid_for_cert() {
        let settings = settings(&Strings {
            pod_name: "foo",
            pod_ns: "ns1",
            controller_ns: "conduit",
            trust_anchors: "ca1.pem",
            end_entity_cert: "foo-ns1-ca1.crt",
            private_key: "bar-ns1-ca1.p8", // Mismatch
        });
        match CommonConfig::load_from_disk(&settings) {
            Err(_) => (), // // TODO: Err(Error::InvalidPrivateKey) > (),
            r => unreachable!("CommonConfig::load_from_disk returned {:?}", r),
        }
    }
}
