package github

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// ResolveToken finds a GitHub token from env vars or the gh CLI.
func ResolveToken() (string, error) {
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		return t, nil
	}
	if t := os.Getenv("GH_TOKEN"); t != "" {
		return t, nil
	}

	out, err := exec.Command("gh", "auth", "token").Output() //nolint:gosec // gh binary is hardcoded, not user input
	if err == nil {
		t := strings.TrimSpace(string(out))
		if t != "" {
			return t, nil
		}
	}

	return "", fmt.Errorf("no GitHub token found; set GITHUB_TOKEN, GH_TOKEN, or run `gh auth login`")
}

// NewClient creates a GitHub GraphQL client authenticated with the given token.
func NewClient(token string) *githubv4.Client {
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := &http.Client{Transport: &oauth2.Transport{Source: src}}
	return githubv4.NewClient(httpClient)
}
