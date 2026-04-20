//go:build !darwin

package github

import "net/http"

func githubHTTPRoundTripper() http.RoundTripper {
	return nil
}
