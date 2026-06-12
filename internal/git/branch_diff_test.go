package git

import (
	"os"
	"path/filepath"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type wantFile struct {
	path   string
	status string
}

func TestBranchDiff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		setup             func(t *testing.T) (dir string, opts BranchDiffOptions)
		wantBase          string
		wantInPatch       []string
		wantFile          *wantFile
		wantEmpty         bool
		checkLocalChanges bool
		wantLocalChanges  bool
	}{
		{
			name: "feature commit ahead of main",
			setup: func(t *testing.T) (string, BranchDiffOptions) {
				dir, repo := initTestRepo(t)
				mainHash := addAndCommit(t, repo, dir, "README.md", "base\n", "init")
				renameDefaultBranch(t, repo, "main")
				setOriginDefault(t, repo, "main", mainHash)
				checkoutBranch(t, repo, "feature", true)
				addAndCommit(t, repo, dir, "feature.txt", "feature line\n", "feature work")
				return dir, BranchDiffOptions{}
			},
			wantBase:    "main",
			wantInPatch: []string{"feature.txt", "+feature line"},
		},
		{
			name: "unstaged worktree change",
			setup: func(t *testing.T) (string, BranchDiffOptions) {
				dir, repo := initTestRepo(t)
				mainHash := addAndCommit(t, repo, dir, "README.md", "base\n", "init")
				renameDefaultBranch(t, repo, "main")
				setOriginDefault(t, repo, "main", mainHash)
				checkoutBranch(t, repo, "feature", true)
				addAndCommit(t, repo, dir, "feature.txt", "committed\n", "feature work")
				writeFile(t, dir, "feature.txt", "committed\nlocal edit\n")
				return dir, BranchDiffOptions{}
			},
			wantBase:    "main",
			wantInPatch: []string{"+local edit"},
		},
		{
			name: "staged change without commit",
			setup: func(t *testing.T) (string, BranchDiffOptions) {
				dir, repo := initTestRepo(t)
				mainHash := addAndCommit(t, repo, dir, "README.md", "base\n", "init")
				renameDefaultBranch(t, repo, "main")
				setOriginDefault(t, repo, "main", mainHash)
				checkoutBranch(t, repo, "feature", true)
				writeFile(t, dir, "staged.txt", "staged content\n")
				w, err := repo.Worktree()
				require.NoError(t, err)
				_, err = w.Add("staged.txt")
				require.NoError(t, err)
				return dir, BranchDiffOptions{}
			},
			wantBase:    "main",
			wantInPatch: []string{"staged.txt", "+staged content"},
		},
		{
			name: "clean on default branch",
			setup: func(t *testing.T) (string, BranchDiffOptions) {
				dir, repo := initTestRepo(t)
				mainHash := addAndCommit(t, repo, dir, "README.md", "base\n", "init")
				renameDefaultBranch(t, repo, "main")
				setOriginDefault(t, repo, "main", mainHash)
				return dir, BranchDiffOptions{}
			},
			wantBase:  "main",
			wantEmpty: true,
		},
		{
			name: "explicit base override",
			setup: func(t *testing.T) (string, BranchDiffOptions) {
				dir, repo := initTestRepo(t)
				addAndCommit(t, repo, dir, "README.md", "base\n", "init")
				addAndCommit(t, repo, dir, "second.txt", "second\n", "second")
				checkoutBranch(t, repo, "feature", true)
				addAndCommit(t, repo, dir, "feature.txt", "feature\n", "feature")
				return dir, BranchDiffOptions{Base: "master"}
			},
			wantBase:    "master",
			wantInPatch: []string{"feature.txt", "+feature"},
		},
		{
			name: "binary file change",
			setup: func(t *testing.T) (string, BranchDiffOptions) {
				dir, repo := initTestRepo(t)
				mainHash := addAndCommitBytes(t, repo, dir, "data.bin", []byte{0x00, 0x01}, "init")
				renameDefaultBranch(t, repo, "main")
				setOriginDefault(t, repo, "main", mainHash)
				checkoutBranch(t, repo, "feature", true)
				addAndCommitBytes(t, repo, dir, "data.bin", []byte{0x00, 0x02}, "update binary")
				return dir, BranchDiffOptions{}
			},
			wantBase: "main",
			wantFile: &wantFile{
				path:   "data.bin",
				status: "binary",
			},
		},
		{
			name: "deleted file on branch",
			setup: func(t *testing.T) (string, BranchDiffOptions) {
				dir, repo := initTestRepo(t)
				mainHash := addAndCommit(t, repo, dir, "keep.txt", "stay\n", "init")
				addAndCommit(t, repo, dir, "remove.txt", "gone\n", "add remove")
				renameDefaultBranch(t, repo, "main")
				setOriginDefault(t, repo, "main", mainHash)
				checkoutBranch(t, repo, "feature", true)
				w, err := repo.Worktree()
				require.NoError(t, err)
				require.NoError(t, os.Remove(filepath.Join(dir, "remove.txt")))
				_, err = w.Add("remove.txt")
				require.NoError(t, err)
				_, err = w.Commit("delete file", &gogit.CommitOptions{Author: testAuthor})
				require.NoError(t, err)
				return dir, BranchDiffOptions{}
			},
			wantBase: "main",
			wantFile: &wantFile{
				path:   "remove.txt",
				status: "deleted",
			},
		},
		{
			name: "local changes on default branch",
			setup: func(t *testing.T) (string, BranchDiffOptions) {
				dir, repo := initTestRepo(t)
				mainHash := addAndCommit(t, repo, dir, "README.md", "base\n", "init")
				renameDefaultBranch(t, repo, "main")
				setOriginDefault(t, repo, "main", mainHash)
				writeFile(t, dir, "README.md", "base\nlocal edit\n")
				return dir, BranchDiffOptions{}
			},
			wantBase:          "main",
			checkLocalChanges: true,
			wantLocalChanges:  true,
			wantInPatch:       []string{"+local edit"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir, opts := tt.setup(t)
			result, err := BranchDiff(dir, opts)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.wantBase, result.BaseBranch)
			if tt.checkLocalChanges {
				assert.Equal(t, tt.wantLocalChanges, result.HasLocalChanges)
			}

			if tt.wantEmpty {
				assert.Empty(t, result.Patch)
				assert.Equal(t, 0, result.Stats.FilesChanged)
				return
			}

			if tt.wantFile != nil {
				var found *FileDiff
				for i := range result.Files {
					if result.Files[i].Path == tt.wantFile.path {
						found = &result.Files[i]
						break
					}
				}
				require.NotNil(t, found, "file %q not in diff", tt.wantFile.path)
				assert.Equal(t, tt.wantFile.status, found.Status)
			}

			for _, want := range tt.wantInPatch {
				assert.Contains(t, result.Patch, want, "patch missing %q:\n%s", want, result.Patch)
			}
		})
	}
}

func TestBranchDiffDetachedHEAD(t *testing.T) {
	t.Parallel()

	dir, repo := initTestRepo(t)
	hash := addAndCommit(t, repo, dir, "README.md", "base\n", "init")
	detachHEAD(t, repo, hash)

	_, err := BranchDiff(dir, BranchDiffOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "detached")
}
