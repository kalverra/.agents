package review

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kalverra/agents/internal/git"
	"github.com/kalverra/agents/internal/github"
)

func TestFormatBundleWithPR(t *testing.T) {
	t.Parallel()

	pr := &github.PR{
		Number:         42,
		Title:          "Fix things",
		URL:            "https://github.com/o/r/pull/42",
		Author:         "alice",
		BaseRef:        "main",
		HeadRef:        "feature",
		ReviewDecision: "REVIEW_REQUIRED",
		Threads: []github.ReviewThread{
			{
				Path: "main.go",
				Line: 10,
				Comments: []github.Comment{
					{Author: "bob", Body: "nit", CreatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
				},
			},
		},
	}

	diff := &git.BranchDiffResult{
		BaseBranch:    "main",
		CurrentBranch: "feature",
		Files: []git.FileDiff{
			{
				Path:      "main.go",
				Status:    "modified",
				Additions: 1,
				Deletions: 1,
				Patch:     "diff --git a/main.go b/main.go\n@@ -1 +1 @@\n-old\n+new\n",
			},
		},
	}

	out := FormatBundle(pr, diff, false)

	assert.Contains(t, out, "# PR #42: Fix things")
	assert.Contains(t, out, "## Unresolved Threads (Not in Diff)")
	assert.Contains(t, out, "main.go:10")
	assert.Contains(t, out, "> nit")
	assert.Contains(t, out, "## Code Diff (main...feature)")
	assert.Contains(t, out, "-old")
	assert.Contains(t, out, "+new")
	assert.NotContains(t, out, "PR diff:")
}

func TestFormatBundleWithoutPR(t *testing.T) {
	t.Parallel()

	diff := &git.BranchDiffResult{
		BaseBranch:    "main",
		CurrentBranch: "feature",
		Files: []git.FileDiff{
			{
				Path:      "main.go",
				Status:    "modified",
				Additions: 1,
				Deletions: 0,
				Patch:     "diff --git a/main.go b/main.go\n@@ -0,0 +1 @@\n+new\n",
			},
		},
	}

	out := FormatBundle(nil, diff, false)

	assert.Contains(t, out, "# Code Review: feature → main")
	assert.Contains(t, out, "## Code Diff (main...feature)")
	assert.Contains(t, out, "+new")
	assert.NotContains(t, out, "## General Comments")
	assert.NotContains(t, out, "## PR Description")
}

func TestFilename(t *testing.T) {
	t.Parallel()

	diff := &git.BranchDiffResult{
		BaseBranch:    "main",
		CurrentBranch: "feature/foo",
	}

	name := Filename(&github.PR{Number: 99}, diff)
	require.Equal(t, "pr-99_review.md", name)

	name = Filename(nil, diff)
	require.Equal(t, "feature-foo_main_review.md", name)
}
