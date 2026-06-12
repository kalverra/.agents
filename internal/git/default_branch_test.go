package git

import (
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectDefaultBranch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(t *testing.T) *gogit.Repository
		want    string
		wantErr bool
	}{
		{
			name: "origin HEAD points to main",
			setup: func(t *testing.T) *gogit.Repository {
				dir, repo := initTestRepo(t)
				mainHash := addAndCommit(t, repo, dir, "README.md", "base\n", "init")
				renameDefaultBranch(t, repo, "main")
				setOriginDefault(t, repo, "main", mainHash)
				checkoutBranch(t, repo, "feature", true)
				addAndCommit(t, repo, dir, "feature.txt", "feature\n", "feature work")
				return repo
			},
			want: "main",
		},
		{
			name: "falls back to master",
			setup: func(t *testing.T) *gogit.Repository {
				dir, repo := initTestRepo(t)
				addAndCommit(t, repo, dir, "README.md", "base\n", "init")
				return repo
			},
			want: "master",
		},
		{
			name: "falls back to local main",
			setup: func(t *testing.T) *gogit.Repository {
				dir, repo := initTestRepo(t)
				addAndCommit(t, repo, dir, "README.md", "base\n", "init")
				renameDefaultBranch(t, repo, "main")
				return repo
			},
			want: "main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := tt.setup(t)
			got, err := DetectDefaultBranch(repo)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
