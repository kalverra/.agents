// Package review bundles PR metadata, review threads, and branch diffs for agent consumption.
package review

import (
	"fmt"
	"strings"

	"github.com/kalverra/agents/internal/git"
	"github.com/kalverra/agents/internal/github"
)

// FormatBundle renders PR review threads and branch diff as a single markdown document.
// pr may be nil when no open PR exists for the current branch.
func FormatBundle(pr *github.PR, diff *git.BranchDiffResult, includeResolved bool) string {
	var b strings.Builder

	if pr != nil {
		github.WritePRHeader(&b, pr)
		github.WritePRReviewSections(&b, pr, includeResolved)
	} else {
		fmt.Fprintf(&b, "# Code Review: %s → %s\n\n", diff.CurrentBranch, diff.BaseBranch)
	}

	fmt.Fprintf(&b, "\n## Code Diff (%s...%s)\n\n", diff.BaseBranch, diff.CurrentBranch)

	rawDiff := git.FormatDiffBody(diff)
	var notInterleaved []github.ReviewThread

	if pr != nil && len(pr.Threads) > 0 {
		var active []github.ReviewThread
		for _, t := range pr.Threads {
			if t.IsResolved && !includeResolved {
				continue
			}
			active = append(active, t)
		}
		rawDiff, notInterleaved = injectComments(rawDiff, active)
	}

	b.WriteString(rawDiff)

	if len(notInterleaved) > 0 {
		fmt.Fprintf(&b, "\n\n## Unresolved Threads (Not in Diff)\n\n")
		for _, t := range notInterleaved {
			loc := t.Path
			if t.Line > 0 {
				loc = fmt.Sprintf("%s:%d", t.Path, t.Line)
			}
			label := "Thread"
			if t.IsOutdated {
				label = "Thread (outdated)"
			}
			fmt.Fprintf(&b, "### %s: %s\n\n", label, loc)

			// Add code snippet
			if t.Line > 0 && diff.RepoPath != "" {
				snippet := git.ReadFileSnippet(diff.RepoPath, t.Path, t.Line, 2)
				if snippet != "" {
					fmt.Fprintf(&b, "```go\n%s\n```\n\n", snippet)
				}
			}

			for _, c := range t.Comments {
				date := c.CreatedAt.Format("2006-01-02")
				fmt.Fprintf(&b, "**@%s** (%s):\n", c.Author, date)
				for line := range strings.SplitSeq(github.StripBloat(c.Body), "\n") {
					fmt.Fprintf(&b, "> %s\n", line)
				}
				b.WriteString("\n")
			}
		}
	}

	return b.String()
}

// Filename returns the output filename for a review bundle.
func Filename(pr *github.PR, diff *git.BranchDiffResult) string {
	if pr != nil {
		return fmt.Sprintf("pr-%d_review.md", pr.Number)
	}
	safeCurrent := strings.ReplaceAll(diff.CurrentBranch, "/", "-")
	safeBase := strings.ReplaceAll(diff.BaseBranch, "/", "-")
	return fmt.Sprintf("%s_%s_review.md", safeCurrent, safeBase)
}
