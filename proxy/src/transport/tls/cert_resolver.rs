use std::{
    fmt,
    sync::Arc,
};

use super::{
    config,

    rustls,
    untrusted,
    webpki,
};
use ring::{self, rand, signature};

pub struct CertResolver {
    certified_key: rustls::sign::CertifiedKey,
}

impl fmt::Debug for CertResolver {
    fn fmt(&self, f: &mut fmt::Formatter) -> Result<(), fmt::Error> {
        f.debug_struct("CertResolver")
            .finish()
    }
}

struct SigningKey {
    signer: Signer,
}

#[derive(Clone)]
struct Signer {
    private_key: Arc<signature::KeyPair>,
}

impl CertResolver {
    /// Returns a new `CertResolver` that has a certificate (chain) verified to
    /// have been issued by one of the given trust anchors.
    ///
    /// TODO: Have the caller pass in a `rustls::ServerCertVerified` as evidence
    /// that the certificate chain was validated, once Rustls's (safe) API
    /// supports that.
    ///
    /// TODO: Verify that the public key of the certificate matches the private
    /// key.
    pub fn new(
        _certificate_was_validated: (), // TODO: `rustls::ServerCertVerified`.
        cert_chain: Vec<rustls::Certificate>,
        private_key: untrusted::Input)
        -> Result<Self, config::Error>
    {
        let private_key = signature::key_pair_from_pkcs8(SIGNATURE_ALG_RING_SIGNING, private_key)
            .map_err(|ring::error::Unspecified| config::Error::InvalidPrivateKey)?;

        let signer = Signer { private_key: Arc::new(private_key) };
        let signing_key = SigningKey { signer };
        let certified_key = rustls::sign::CertifiedKey::new(
            cert_chain, Arc::new(Box::new(signing_key)));
        Ok(Self { certified_key })
    }
}

fn parse_end_entity_cert<'a>(cert_chain: &'a[rustls::Certificate])
    -> Result<webpki::EndEntityCert<'a>, webpki::Error>
{
    let cert = cert_chain.first()
        .map(rustls::Certificate::as_ref)
        .unwrap_or(&[]); // An empty input fill fail to parse.
    webpki::EndEntityCert::from(untrusted::Input::from(cert))
}

impl rustls::ResolvesServerCert for CertResolver {
    fn resolve(&self, server_name: Option<webpki::DNSNameRef>,
               sigschemes: &[rustls::SignatureScheme]) -> Option<rustls::sign::CertifiedKey> {
        let server_name = if let Some(server_name) = server_name {
            server_name
        } else {
            debug!("no SNI -> no certificate");
            return None;
        };

        if !sigschemes.contains(&SIGNATURE_ALG_RUSTLS_SCHEME) {
            debug!("signature scheme not supported -> no certificate");
            return None;
        }

        // Verify that our certificate is valid for the given SNI name.
        if let Err(err) = parse_end_entity_cert(&self.certified_key.cert)
            .and_then(|cert| cert.verify_is_valid_for_dns_name(server_name)) {
            debug!("our certificate is not valid for the SNI name -> no certificate: {:?}", err);
            return None;
        }

        Some(self.certified_key.clone())
    }
}

impl rustls::sign::SigningKey for SigningKey {
    fn choose_scheme(&self, offered: &[rustls::SignatureScheme])
        -> Option<Box<rustls::sign::Signer>>
    {
        if offered.contains(&SIGNATURE_ALG_RUSTLS_SCHEME) {
            Some(Box::new(self.signer.clone()))
        } else {
            None
        }
    }

    fn algorithm(&self) -> rustls::internal::msgs::enums::SignatureAlgorithm {
        SIGNATURE_ALG_RUSTLS_ALGORITHM
    }
}

impl rustls::sign::Signer for Signer {
    fn sign(&self, message: &[u8]) -> Result<Vec<u8>, rustls::TLSError> {
        let rng = rand::SystemRandom::new();
        signature::sign(&self.private_key, &rng, untrusted::Input::from(message))
            .map(|signature| signature.as_ref().to_owned())
            .map_err(|ring::error::Unspecified|
                rustls::TLSError::General("Signing Failed".to_owned()))
    }

    fn get_scheme(&self) -> rustls::SignatureScheme {
        SIGNATURE_ALG_RUSTLS_SCHEME
    }
}

// Keep these in sync.
static SIGNATURE_ALG_RING_SIGNING: &signature::SigningAlgorithm =
    &signature::ECDSA_P256_SHA256_ASN1_SIGNING;
const SIGNATURE_ALG_RUSTLS_SCHEME: rustls::SignatureScheme =
    rustls::SignatureScheme::ECDSA_NISTP256_SHA256;
const SIGNATURE_ALG_RUSTLS_ALGORITHM: rustls::internal::msgs::enums::SignatureAlgorithm =
    rustls::internal::msgs::enums::SignatureAlgorithm::ECDSA;
