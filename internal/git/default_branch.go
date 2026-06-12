package git

import (
	"fmt"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

var defaultBranchCandidates = []string{"main", "master"}

// DetectDefaultBranch resolves the repository default branch name.
func DetectDefaultBranch(repo *gogit.Repository) (string, error) {
	if name, err := defaultBranchFromOriginHEAD(repo); err == nil && name != "" {
		return name, nil
	}

	if name, err := defaultBranchFromUpstream(repo); err == nil && name != "" {
		return name, nil
	}

	for _, candidate := range defaultBranchCandidates {
		if hasBranchRef(repo, plumbing.NewBranchReferenceName(candidate)) {
			return candidate, nil
		}
	}

	for _, candidate := range defaultBranchCandidates {
		remoteRef := plumbing.NewRemoteReferenceName("origin", candidate)
		if _, err := repo.Reference(remoteRef, false); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("could not detect default branch; set one with --base")
}

func defaultBranchFromOriginHEAD(repo *gogit.Repository) (string, error) {
	ref, err := repo.Reference(plumbing.NewRemoteHEADReferenceName("origin"), true)
	if err != nil {
		return "", err
	}

	target := ref.Target()
	if !target.IsRemote() {
		return "", fmt.Errorf("origin HEAD does not point to a remote branch")
	}

	return strings.TrimPrefix(target.Short(), "origin/"), nil
}

func defaultBranchFromUpstream(repo *gogit.Repository) (string, error) {
	head, err := repo.Head()
	if err != nil {
		return "", err
	}
	if !head.Name().IsBranch() {
		return "", fmt.Errorf("HEAD is not a branch")
	}

	cfg, err := repo.Config()
	if err != nil {
		return "", err
	}

	branchCfg, ok := cfg.Branches[head.Name().Short()]
	if !ok || branchCfg.Merge == "" {
		return "", fmt.Errorf("no upstream configured for %s", head.Name().Short())
	}

	return branchCfg.Merge.Short(), nil
}

func hasBranchRef(repo *gogit.Repository, name plumbing.ReferenceName) bool {
	_, err := repo.Reference(name, false)
	return err == nil
}
