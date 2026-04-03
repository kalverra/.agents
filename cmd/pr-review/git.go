package main

import (
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
)

type RepoInfo struct {
	Owner  string
	Name   string
	Branch string
}

var sshRemoteRe = regexp.MustCompile(`^[\w.-]+@[\w.-]+:([\w.-]+)/([\w.-]+?)(?:\.git)?$`)

func DetectRepo(dir string) (*RepoInfo, error) {
	if err := runGit(dir, "rev-parse", "--show-toplevel"); err != nil {
		return nil, fmt.Errorf("not a git repository (or any parent): %w", err)
	}

	remote, err := gitOutput(dir, "remote", "get-url", "origin")
	if err != nil {
		return nil, fmt.Errorf("no 'origin' remote configured: %w", err)
	}

	owner, name, err := parseRemoteURL(remote)
	if err != nil {
		return nil, err
	}

	branch, err := gitOutput(dir, "branch", "--show-current")
	if err != nil {
		return nil, fmt.Errorf("could not determine current branch: %w", err)
	}
	if branch == "" {
		return nil, fmt.Errorf("HEAD is detached; check out a branch first")
	}

	return &RepoInfo{Owner: owner, Name: name, Branch: branch}, nil
}

func parseRemoteURL(remote string) (owner, name string, err error) {
	remote = strings.TrimSpace(remote)

	if m := sshRemoteRe.FindStringSubmatch(remote); m != nil {
		return m[1], m[2], nil
	}

	u, parseErr := url.Parse(remote)
	if parseErr == nil && u.Host != "" {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) >= 2 {
			repo := strings.TrimSuffix(parts[1], ".git")
			return parts[0], repo, nil
		}
	}

	return "", "", fmt.Errorf("cannot parse remote URL %q into owner/repo", remote)
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.Run()
}
