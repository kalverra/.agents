//go:build darwin

package github

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

func githubHTTPRoundTripper() http.RoundTripper {
	// Default: pure-Go verification with public roots (works under sandbox).
	// Set AGENTS_TLS_MACOS_USE_KEYCHAIN=1 to use the system HTTP transport and
	// Keychain-based verification (e.g. enterprise CAs only in the keychain).
	if os.Getenv("AGENTS_TLS_MACOS_USE_KEYCHAIN") == "1" {
		return nil
	}

	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(mozillaCACertPEM); !ok {
		panic("github: mozilla CA bundle did not parse")
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
