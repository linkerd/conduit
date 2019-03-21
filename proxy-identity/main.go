package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/linkerd/linkerd2/pkg/flags"
	"github.com/linkerd/linkerd2/pkg/tls"
	log "github.com/sirupsen/logrus"
)

const (
	envDisabled     = "LINKERD2_PROXY_IDENTITY_DISABLED"
	envTrustAnchors = "LINKERD2_PROXY_IDENTITY_TRUST_ANCHORS"
)

func main() {
	name := flag.String("name", "", "identity name")
	dir := flag.String("dir", "", "directory under which credentials are written")
	flags.ConfigureAndParse()

	if os.Getenv(envDisabled) != "" {
		log.Debug("Identity disabled.")
		os.Exit(0)
	}

	keyPath, csrPath, err := checkEndEntityDir(*dir)
	if err != nil {
		log.Fatalf("Invalid end-entity directory: %s", err)
	}

	if _, err := loadVerifier(os.Getenv(envTrustAnchors)); err != nil {
		log.Fatalf("Failed to load trust anchors: %s", err)
	}

	key, err := generateAndStoreKey(keyPath)
	if err != nil {
		log.Fatal(err.Error())
	}

	if _, err := generateAndStoreCSR(csrPath, *name, key); err != nil {
		log.Fatal(err.Error())
	}
}

func loadVerifier(pem string) (verify x509.VerifyOptions, err error) {
	if pem == "" {
		err = fmt.Errorf("%s must be set", envTrustAnchors)
		return
	}

	verify.Roots, err = tls.DecodePEMCertPool(pem)
	return
}

// checkEndEntityDir checks that the provided directory path exists and is
// suitable to write key material to, returning the Key, CSR, and Crt paths.
//
// If the directory does not exist or if it has incorrect permissions, we assume
// that the wrong directory was specified incorrectly, instead of trying to
// create or repair the directory. In practice, this directory should be tmpfs
// so that credentials are not written to disk, so we want to be extra sensitive
// to an incorrectly specified path.
//
// If the key, CSR, and/or Crt paths refer to existing files, it is assumed that
// multiple instances of this process are running, and an error is returned.
func checkEndEntityDir(dir string) (string, string, error) {
	if dir == "" {
		return "", "", errors.New("No end entity directory specified")
	}

	s, err := os.Stat(dir)
	if err != nil {
		return "", "", err
	}
	if !s.IsDir() {
		return "", "", fmt.Errorf("Not a directory: %s", dir)
	}
	// if s.Mode().Perm()&0002 == 0002 {
	// 	return "", "", "", fmt.Errorf("Must not be world-writeable: %s; got %s", dir, s.Mode().Perm())
	// }

	keyPath := filepath.Join(dir, "key.p8")
	if err = checkNotExists(keyPath); err != nil {
		log.Info(err.Error())
	}

	csrPath := filepath.Join(dir, "csr.der")
	if err = checkNotExists(csrPath); err != nil {
		log.Info(err.Error())
	}

	return keyPath, csrPath, nil
}

func checkNotExists(p string) (err error) {
	_, err = os.Stat(p)
	if err == nil {
		err = fmt.Errorf("Already exists: %s", p)
	} else if os.IsNotExist(err) {
		err = nil
	}
	return
}

func generateAndStoreKey(p string) (key *ecdsa.PrivateKey, err error) {
	// Generate a private key and store it read-only (i.e. mostly for debugging). Because the file is read-only
	key, err = tls.GenerateKey()
	if err != nil {
		return
	}

	pemb := tls.EncodePrivateKeyP8(key)
	err = ioutil.WriteFile(p, pemb, 0600)
	return
}

func generateAndStoreCSR(p, id string, key *ecdsa.PrivateKey) ([]byte, error) {
	if id == "" {
		return nil, errors.New("A non-empty identity is required")
	}

	// TODO do proper DNS name validation.
	csr := x509.CertificateRequest{DNSNames: []string{id}}
	csrb, err := x509.CreateCertificateRequest(rand.Reader, &csr, key)
	if err != nil {
		return nil, fmt.Errorf("Failed to create CSR: %s", err)
	}

	if err = ioutil.WriteFile(p, csrb, 0600); err != nil {
		return nil, fmt.Errorf("Failed to write CSR: %s", err)
	}

	return csrb, nil
}
