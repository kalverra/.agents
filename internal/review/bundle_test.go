package review

import (
	"strings"
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

	assert.Contains(t, out, `<pr id="42"`)
	assert.Contains(t, out, `<unresolved_threads>`)
	assert.Contains(t, out, `<thread line="10" author="bob">nit</thread>`)
	assert.Contains(t, out, `<file path="main.go" status="modified" additions="1" deletions="1">`)
	assert.Contains(t, out, "-old")
	assert.Contains(t, out, "+new")
}

func TestFormatBundleWithSuggestionAndDiffHunk(t *testing.T) {
	t.Parallel()

	pr := &github.PR{
		Number:  22873,
		Author:  "alice",
		BaseRef: "develop",
		HeadRef: "feature",
		Threads: []github.ReviewThread{
			{
				Path:      "core/engine.go",
				StartLine: 1174,
				Line:      1178,
				Comments: []github.Comment{
					{
						Author: "Copilot",
						Body:   "keep a deadline after WithoutCancel",
						Suggestions: []github.Suggestion{
							{
								Source: github.SuggestionSourceAutomated,
								Code:   "ctx = context.WithoutCancel(ctx)\nif deadline, ok := ctx.Deadline(); ok {\n\tctx, _ = context.WithDeadline(ctx, deadline)\n}",
							},
						},
						DiffHunk: "@@ -10 +10 @@\n+ctx = context.WithoutCancel(ctx)",
					},
				},
			},
		},
	}

	diff := &git.BranchDiffResult{
		BaseBranch:    "develop",
		CurrentBranch: "feature",
		Files: []git.FileDiff{
			{
				Path:      "core/engine.go",
				Status:    "modified",
				Additions: 1,
				Deletions: 0,
				Patch:     "diff --git a/core/engine.go b/core/engine.go\n@@ -1174,5 +1174,6 @@\n+ctx = context.WithoutCancel(ctx)\n",
			},
		},
	}

	out := FormatBundle(pr, diff, false)

	assert.Contains(t, out, `start_line="1174"`)
	assert.Contains(t, out, `<suggestion source="automated"`)
	assert.Contains(t, out, "context.WithDeadline")
	assert.Contains(t, out, "<diff_hunk>")
	assert.Contains(t, out, "context.WithoutCancel(ctx)")
}

func TestFormatBundleInterleavedThreadOmitsDiffHunk(t *testing.T) {
	t.Parallel()

	sharedHunk := "@@ -1,4 +1,5 @@\n context\n+added line"
	pr := &github.PR{
		Threads: []github.ReviewThread{
			{
				Path: "main.go",
				Line: 5,
				Comments: []github.Comment{
					{Author: "bolekk", Body: "first concern", DiffHunk: sharedHunk},
					{Author: "bolekk", Body: "follow-up thought", DiffHunk: sharedHunk},
					{Author: "bolekk", Body: "alternative idea", DiffHunk: sharedHunk},
				},
			},
		},
	}

	diff := &git.BranchDiffResult{
		Files: []git.FileDiff{
			{
				Path:   "main.go",
				Status: "modified",
				Patch: "diff --git a/main.go b/main.go\n" +
					"@@ -1,4 +1,5 @@\n" +
					" line1\n" +
					" line2\n" +
					" line3\n" +
					" line4\n" +
					"+added line\n",
			},
		},
	}

	out := FormatBundle(pr, diff, false)

	assert.Equal(t, 3, strings.Count(out, "<comment "))
	assert.NotContains(t, out, "<diff_hunk>")
	assert.NotContains(t, out, "<unresolved_threads>")
}

func TestFormatBundleNonInterleavedThreadDiffHunkOnce(t *testing.T) {
	t.Parallel()

	sharedHunk := "@@ -99 +99 @@\n+orphan"
	pr := &github.PR{
		Threads: []github.ReviewThread{
			{
				Path: "other.go",
				Line: 10,
				Comments: []github.Comment{
					{Author: "bob", Body: "one", DiffHunk: sharedHunk},
					{Author: "bob", Body: "two", DiffHunk: sharedHunk},
				},
			},
		},
	}

	diff := &git.BranchDiffResult{
		Files: []git.FileDiff{
			{Path: "main.go", Status: "modified", Patch: "diff --git a/main.go b/main.go\n@@ -1 +1 @@\n-old\n+new\n"},
		},
	}

	out := FormatBundle(pr, diff, false)

	assert.Equal(t, 1, strings.Count(out, "<diff_hunk>"))
	assert.Equal(t, 2, strings.Count(out, "<comment "))
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

	assert.Contains(t, out, `<pr branch="feature -> main">`)
	assert.Contains(t, out, `<diff>`)
	assert.Contains(t, out, `<hunk line="1">`)
	assert.Contains(t, out, "+new")
}

func TestFilename(t *testing.T) {
	t.Parallel()

	diff := &git.BranchDiffResult{
		BaseBranch:    "main",
		CurrentBranch: "feature/foo",
	}

	name := Filename(&github.PR{Number: 99}, diff)
	require.Equal(t, "pr-99_review.xml", name)

	name = Filename(nil, diff)
	require.Equal(t, "feature-foo_main_review.xml", name)
}
