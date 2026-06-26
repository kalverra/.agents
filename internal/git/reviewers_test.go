package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuggestReviewers(t *testing.T) {
	t.Parallel()

	dir := initTestRepo(t)

	// Create initial file with Author 1
	runGitCmd(t, dir, "config", "user.name", "Author One")
	runGitCmd(t, dir, "config", "user.email", "one@example.com")
	mainHash := addAndCommit(t, dir, "README.md", "line 1\nline 2\nline 3\nline 4\nline 5\n", "init")
	renameDefaultBranch(t, dir, "main")
	setOriginDefault(t, dir, "main", mainHash)

	// Author 2 modifies some lines
	runGitCmd(t, dir, "config", "user.name", "Author Two")
	runGitCmd(t, dir, "config", "user.email", "two@example.com")
	addAndCommit(t, dir, "README.md", "line 1\nline 2 mod\nline 3\nline 4\nline 5\n", "mod by two")

	// Author 3 modifies some lines
	runGitCmd(t, dir, "config", "user.name", "Author Three")
	runGitCmd(t, dir, "config", "user.email", "three@example.com")
	baseHash := addAndCommit(t, dir, "README.md", "line 1\nline 2 mod\nline 3 mod\nline 4\nline 5\n", "mod by three")

	// Now we are Author Four, making changes in a PR
	runGitCmd(t, dir, "config", "user.name", "Author Four")
	runGitCmd(t, dir, "config", "user.email", "four@example.com")

	// Simulate diff where we modify line 2 and 3
	patch := `diff --git a/README.md b/README.md
index 1234567..89abcdef 100644
--- a/README.md
+++ b/README.md
@@ -1,5 +1,5 @@
 line 1
-line 2 mod
-line 3 mod
+line 2 new
+line 3 new
 line 4
 line 5
`

	diffResult := &BranchDiffResult{
		MergeBase: baseHash,
		Files: []FileDiff{
			{
				Path:  "README.md",
				Patch: patch,
			},
		},
	}

	reviewers, err := SuggestReviewers(dir, diffResult, "Author Four", 3)
	require.NoError(t, err)

	// Line 2 was last modified by Author Two
	// Line 3 was last modified by Author Three
	// So both should be suggested. Author One only touched line 1, 4, 5, which are context lines.
	// But we blame on deleted lines (-2,2) -> lines 2 and 3.

	assert.ElementsMatch(t, []string{"Author Three", "Author Two"}, reviewers)
}
