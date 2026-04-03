package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

func main() {
	dir := flag.String("dir", ".", "Git repository directory")
	includeResolved := flag.Bool("include-resolved", false, "Show resolved threads in full")
	flag.Parse()

	token, err := resolveToken()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	repo, err := DetectRepo(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	client := newGitHubClient(token)
	ctx := context.Background()

	pr, err := FetchPR(ctx, client, repo.Owner, repo.Name, repo.Branch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if pr == nil {
		fmt.Printf("No open PR found for branch %q in %s/%s.\n", repo.Branch, repo.Owner, repo.Name)
		os.Exit(0)
	}

	fmt.Print(FormatPR(pr, *includeResolved))
}

func resolveToken() (string, error) {
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		return t, nil
	}

	out, err := exec.Command("gh", "auth", "token").Output()
	if err == nil {
		t := strings.TrimSpace(string(out))
		if t != "" {
			return t, nil
		}
	}

	return "", fmt.Errorf(
		"no GitHub token found.\n" +
			"Set GITHUB_TOKEN or run `gh auth login`.",
	)
}

func newGitHubClient(token string) *githubv4.Client {
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := &http.Client{Transport: &oauth2.Transport{Source: src}}
	return githubv4.NewClient(httpClient)
}
