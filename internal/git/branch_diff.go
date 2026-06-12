package git

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// BranchDiffOptions configures BranchDiff.
type BranchDiffOptions struct {
	Base         string
	ContextLines int
}

// DiffStats summarizes patch size.
type DiffStats struct {
	FilesChanged int `json:"files_changed"`
	Insertions   int `json:"insertions"`
	Deletions    int `json:"deletions"`
}

// FileDiff is a per-file patch entry.
type FileDiff struct {
	Path      string `json:"path"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Patch     string `json:"patch,omitempty"`
}

// BranchDiffResult is the normalized branch diff payload.
type BranchDiffResult struct {
	RepoPath        string     `json:"repo_path"`
	CurrentBranch   string     `json:"current_branch"`
	BaseBranch      string     `json:"base_branch"`
	MergeBase       string     `json:"merge_base"`
	Head            string     `json:"head"`
	HasLocalChanges bool       `json:"has_local_changes"`
	Stats           DiffStats  `json:"stats"`
	Files           []FileDiff `json:"files"`
	Patch           string     `json:"patch"`
}

// BranchDiff returns all changes since merge-base with the base branch, including worktree changes.
func BranchDiff(dir string, opts BranchDiffOptions) (*BranchDiffResult, error) {
	repo, err := gogit.PlainOpenWithOptions(dir, &gogit.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, fmt.Errorf("opening repository: %w", err)
	}

	repoPath, err := repoRoot(repo, dir)
	if err != nil {
		return nil, err
	}

	headRef, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("reading HEAD: %w", err)
	}
	if !headRef.Name().IsBranch() {
		return nil, fmt.Errorf("HEAD is detached; check out a branch first")
	}

	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("reading HEAD commit: %w", err)
	}

	baseBranch := opts.Base
	if baseBranch == "" {
		baseBranch, err = DetectDefaultBranch(repo)
		if err != nil {
			return nil, err
		}
	}

	baseCommit, err := resolveBranchCommit(repo, baseBranch)
	if err != nil {
		return nil, fmt.Errorf("resolving base branch %q: %w", baseBranch, err)
	}

	mergeBases, err := headCommit.MergeBase(baseCommit)
	if err != nil {
		return nil, fmt.Errorf("finding merge base: %w", err)
	}
	if len(mergeBases) == 0 {
		return nil, fmt.Errorf("no merge base between %s and %s", headRef.Name().Short(), baseBranch)
	}
	mergeBase := mergeBases[0]

	baseTree, err := mergeBase.Tree()
	if err != nil {
		return nil, fmt.Errorf("reading merge-base tree: %w", err)
	}

	headTree, err := headCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("reading HEAD tree: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("reading worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return nil, fmt.Errorf("reading worktree status: %w", err)
	}

	idx, err := repo.Storer.Index()
	if err != nil {
		return nil, fmt.Errorf("reading index: %w", err)
	}

	idxMap := indexMap(idx)
	paths, err := collectPaths(baseTree, headTree, status)
	if err != nil {
		return nil, err
	}

	fileDiffs := make([]FileDiff, 0, len(paths))

	var patchBuf strings.Builder
	var stats DiffStats

	for _, path := range paths {
		oldSnap, err := treeFileSnapshot(baseTree, path)
		if err != nil {
			return nil, fmt.Errorf("reading merge-base file %q: %w", path, err)
		}

		newSnap, err := effectiveFileSnapshot(repo, repoPath, headTree, idxMap, status, path)
		if err != nil {
			return nil, fmt.Errorf("reading current file %q: %w", path, err)
		}

		if oldSnap.equal(newSnap) {
			continue
		}

		if oldSnap.binary || newSnap.binary {
			fileDiffs = append(fileDiffs, FileDiff{
				Path:   path,
				Status: fileStatusLabel(oldSnap.exists, newSnap.exists, true),
			})
			stats.FilesChanged++
			continue
		}

		filePatch, additions, deletions := unifiedPatch(path, oldSnap.text, newSnap.text, opts.ContextLines)
		fileDiffs = append(fileDiffs, FileDiff{
			Path:      path,
			Status:    fileStatusLabel(oldSnap.exists, newSnap.exists, false),
			Additions: additions,
			Deletions: deletions,
			Patch:     filePatch,
		})
		patchBuf.WriteString(filePatch)
		stats.FilesChanged++
		stats.Insertions += additions
		stats.Deletions += deletions
	}

	return &BranchDiffResult{
		RepoPath:        repoPath,
		CurrentBranch:   headRef.Name().Short(),
		BaseBranch:      baseBranch,
		MergeBase:       shortHash(mergeBase.Hash),
		Head:            shortHash(headRef.Hash()),
		HasLocalChanges: headCommit.Hash == mergeBase.Hash && stats.FilesChanged > 0,
		Stats:           stats,
		Files:           fileDiffs,
		Patch:           patchBuf.String(),
	}, nil
}

func repoRoot(repo *gogit.Repository, dir string) (string, error) {
	wt, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("reading worktree root: %w", err)
	}

	root := wt.Filesystem.Root()
	if root != "" {
		abs, err := filepath.Abs(root)
		if err != nil {
			return "", fmt.Errorf("abs worktree root: %w", err)
		}
		return abs, nil
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("abs dir: %w", err)
	}
	return abs, nil
}

func joinRepoPath(repoPath, path string) string {
	return filepath.Join(repoPath, filepath.FromSlash(path))
}

func resolveBranchCommit(repo *gogit.Repository, branch string) (*object.Commit, error) {
	candidates := []plumbing.ReferenceName{
		plumbing.NewBranchReferenceName(branch),
		plumbing.NewRemoteReferenceName("origin", branch),
	}

	for _, name := range candidates {
		ref, err := repo.Reference(name, false)
		if err != nil {
			continue
		}
		return repo.CommitObject(ref.Hash())
	}

	return nil, fmt.Errorf("branch %q not found", branch)
}

func collectPaths(baseTree, headTree *object.Tree, status gogit.Status) ([]string, error) {
	seen := make(map[string]struct{})

	add := func(path string) {
		if path != "" {
			seen[path] = struct{}{}
		}
	}

	changes, err := object.DiffTree(baseTree, headTree)
	if err != nil {
		return nil, fmt.Errorf("diffing trees: %w", err)
	}
	for _, ch := range changes {
		if ch == nil {
			continue
		}
		if ch.To.Name != "" {
			add(ch.To.Name)
			continue
		}
		add(ch.From.Name)
	}

	for path, st := range status {
		if st.Staging != gogit.Unmodified || st.Worktree != gogit.Unmodified {
			add(path)
		}
	}

	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func isBinaryBytes(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	return bytes.IndexByte(data, 0) >= 0
}

func fileStatusLabel(existed, existsNow, binary bool) string {
	if binary {
		return "binary"
	}
	switch {
	case existed && existsNow:
		return "modified"
	case !existed && existsNow:
		return "added"
	case existed && !existsNow:
		return "deleted"
	default:
		return "changed"
	}
}

func shortHash(hash plumbing.Hash) string {
	s := hash.String()
	if len(s) >= 7 {
		return s[:7]
	}
	return s
}
