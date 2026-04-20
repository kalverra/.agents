//go:build darwin

package mactls

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"net/http"
	"os"
)

// Mozilla CA bundle from https://curl.se/docs/caextract.html (PEM). Public trust
// anchors for TLS — verification uses Go's verifier instead of Security.framework,
// which fails in sandboxed agent environments (x509: OSStatus -26276).
//
//go:embed cacert.pem
var mozillaCACertPEM []byte

// RoundTripper returns a transport with embedded public roots, or nil to use
// [http.DefaultTransport] (Security.framework / Keychain verification).
//
// Set AGENTS_TLS_MACOS_USE_KEYCHAIN=1 to force nil. SSL_CERT_FILE, when set,
// merges additional PEM roots into the pool.
func RoundTripper() http.RoundTripper {
	if os.Getenv("AGENTS_TLS_MACOS_USE_KEYCHAIN") == "1" {
		return nil
	}

	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(mozillaCACertPEM); !ok {
		panic("mactls: mozilla CA bundle did not parse")
	}
	if extra := os.Getenv("SSL_CERT_FILE"); extra != "" {
		if data, err := os.ReadFile(extra); err == nil { //nolint:gosec // path from env, user-controlled by design
			pool.AppendCertsFromPEM(data)
		}
	}

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    pool,
	}
	return tr
}
