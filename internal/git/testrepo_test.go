package git

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/require"
)

var testAuthor = &object.Signature{
	Name:  "Test Author",
	Email: "test@example.com",
	When:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
}

func initTestRepo(t *testing.T) (dir string, repo *gogit.Repository) {
	t.Helper()

	dir = t.TempDir()
	repo, err := gogit.PlainInit(dir, false)
	require.NoError(t, err)

	return dir, repo
}

func writeFile(t *testing.T, dir, path, content string) {
	t.Helper()

	full := filepath.Join(dir, path)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o750))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o600))
}

func addAndCommit(t *testing.T, repo *gogit.Repository, dir, path, content, message string) plumbing.Hash {
	t.Helper()
	return addAndCommitBytes(t, repo, dir, path, []byte(content), message)
}

func addAndCommitBytes(
	t *testing.T,
	repo *gogit.Repository,
	dir, path string,
	content []byte,
	message string,
) plumbing.Hash {
	t.Helper()

	full := filepath.Join(dir, path)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o750))
	require.NoError(t, os.WriteFile(full, content, 0o600))

	w, err := repo.Worktree()
	require.NoError(t, err)

	_, err = w.Add(path)
	require.NoError(t, err)

	hash, err := w.Commit(message, &gogit.CommitOptions{Author: testAuthor})
	require.NoError(t, err)

	return hash
}

func detachHEAD(t *testing.T, repo *gogit.Repository, hash plumbing.Hash) {
	t.Helper()

	ref := plumbing.NewHashReference(plumbing.HEAD, hash)
	require.NoError(t, repo.Storer.SetReference(ref))
}

func checkoutBranch(t *testing.T, repo *gogit.Repository, name string, create bool) {
	t.Helper()

	w, err := repo.Worktree()
	require.NoError(t, err)

	err = w.Checkout(&gogit.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(name),
		Create: create,
	})
	require.NoError(t, err)
}

func renameDefaultBranch(t *testing.T, repo *gogit.Repository, name string) plumbing.Hash {
	t.Helper()

	head, err := repo.Head()
	require.NoError(t, err)

	newRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName(name), head.Hash())
	require.NoError(t, repo.Storer.SetReference(newRef))
	require.NoError(t, repo.Storer.RemoveReference(plumbing.Master))

	sym := plumbing.NewSymbolicReference(plumbing.HEAD, newRef.Name())
	require.NoError(t, repo.Storer.SetReference(sym))

	return head.Hash()
}

func setOriginDefault(t *testing.T, repo *gogit.Repository, branch string, hash plumbing.Hash) {
	t.Helper()

	remoteRef := plumbing.NewHashReference(plumbing.NewRemoteReferenceName("origin", branch), hash)
	require.NoError(t, repo.Storer.SetReference(remoteRef))

	sym := plumbing.NewSymbolicReference(
		plumbing.NewRemoteHEADReferenceName("origin"),
		remoteRef.Name(),
	)
	require.NoError(t, repo.Storer.SetReference(sym))
}
