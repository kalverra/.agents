// Package review formats PR review threads and branch diffs for LLM consumption.
package review

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/kalverra/agents/internal/git"
	"github.com/kalverra/agents/internal/github"
)

var hunkHeaderRegex = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

// FormatBundle renders PR review threads and branch diff as an XML document.
// pr may be nil when no open PR exists for the current branch.
func FormatBundle(pr *github.PR, diff *git.BranchDiffResult, includeResolved bool, suggestedReviewers []string) string {
	var b strings.Builder

	if pr != nil {
		fmt.Fprintf(
			&b,
			"<pr id=\"%d\" author=\"%s\" status=\"%s\" url=\"%s\">\n",
			pr.Number,
			pr.Author,
			pr.ReviewDecision,
			pr.URL,
		)
		fmt.Fprintf(&b, "<branch>%s -> %s</branch>\n", pr.HeadRef, pr.BaseRef)
		b.WriteString("<description>\n")
		if pr.Body != "" {
			b.WriteString(github.StripBloat(pr.Body))
			b.WriteString("\n")
		} else {
			b.WriteString("*(no description)*\n")
		}
		b.WriteString("</description>\n")
	} else {
		fmt.Fprintf(&b, "<pr branch=\"%s -> %s\">\n", diff.CurrentBranch, diff.BaseBranch)
	}

	var codeAdd, codeDel, codeCount int
	var toolAdd, toolDel, toolCount int
	for _, f := range diff.Files {
		if git.IsCodeFile(f.Path) {
			codeCount++
			codeAdd += f.Additions
			codeDel += f.Deletions
		} else {
			toolCount++
			toolAdd += f.Additions
			toolDel += f.Deletions
		}
	}

	b.WriteString("<stats ")
	if codeCount > 0 {
		fmt.Fprintf(&b, "code=\"+%d -%d (%d files)\" ", codeAdd, codeDel, codeCount)
	}
	if toolCount > 0 {
		fmt.Fprintf(&b, "tooling=\"+%d -%d (%d files)\" ", toolAdd, toolDel, toolCount)
	}
	b.WriteString("/>\n\n")

	if len(suggestedReviewers) > 0 {
		b.WriteString("<suggested_reviewers>\n")
		for _, r := range suggestedReviewers {
			fmt.Fprintf(&b, "- %s\n", r)
		}
		b.WriteString("</suggested_reviewers>\n\n")
	}

	b.WriteString("<diff>\n")

	threadMap := make(map[string]map[int][]*github.ReviewThread)
	var notInterleaved []github.ReviewThread

	if pr != nil {
		for i := range pr.Threads {
			t := &pr.Threads[i]
			if t.IsResolved && !includeResolved {
				continue
			}
			if t.Line == 0 {
				notInterleaved = append(notInterleaved, *t)
				continue
			}
			if threadMap[t.Path] == nil {
				threadMap[t.Path] = make(map[int][]*github.ReviewThread)
			}
			threadMap[t.Path][t.Line] = append(threadMap[t.Path][t.Line], t)
		}
	}

	for _, f := range diff.Files {
		fmt.Fprintf(
			&b,
			"<file path=%q status=%q additions=\"%d\" deletions=\"%d\">\n",
			f.Path,
			f.Status,
			f.Additions,
			f.Deletions,
		)

		if f.Patch != "" {
			switch {
			case git.IsDependencyOrLockfile(f.Path):
				b.WriteString("<skipped reason=\"dependency/lockfile/generated\" />\n")
			case !git.IsCodeFile(f.Path) && f.Status == "added" && f.Additions > 40:
				fmt.Fprintf(&b, "<skipped reason=\"%d-line config file\" />\n", f.Additions)
			default:
				formatXMLPatch(&b, f.Patch, threadMap[f.Path])
			}
		}

		b.WriteString("</file>\n\n")
	}

	b.WriteString("</diff>\n")

	// Collect any threads in the map that were not interleaved
	for _, threadsByLine := range threadMap {
		for _, threads := range threadsByLine {
			for _, t := range threads {
				notInterleaved = append(notInterleaved, *t)
			}
		}
	}

	if len(notInterleaved) > 0 {
		b.WriteString("\n<unresolved_threads>\n")
		for i := range notInterleaved {
			writeXMLThread(&b, &notInterleaved[i], true)
		}
		b.WriteString("</unresolved_threads>\n")
	}

	b.WriteString("</pr>\n")
	return b.String()
}

func formatXMLPatch(b *strings.Builder, patch string, threads map[int][]*github.ReviewThread) {
	currentLine := 0
	inHunk := false

	for line := range strings.SplitSeq(patch, "\n") {
		if git.SkipPatchLine(line) {
			continue
		}

		if m := hunkHeaderRegex.FindStringSubmatch(line); m != nil {
			if inHunk {
				b.WriteString("</hunk>\n")
			}
			currentLine, _ = strconv.Atoi(m[1])
			inHunk = true
			fmt.Fprintf(b, "<hunk line=\"%d\">\n", currentLine)
			continue
		}

		if !inHunk {
			continue
		}

		b.WriteString(line)
		b.WriteByte('\n')

		if len(line) > 0 {
			char := line[0]
			switch char {
			case ' ', '+':
				if ts, ok := threads[currentLine]; ok {
					for _, t := range ts {
						writeXMLThread(b, t, false)
					}
					delete(threads, currentLine)
				}
				currentLine++
			case '-':
				// no advance
			}
		} else {
			if ts, ok := threads[currentLine]; ok {
				for _, t := range ts {
					writeXMLThread(b, t, false)
				}
				delete(threads, currentLine)
			}
			currentLine++
		}
	}
	if inHunk {
		b.WriteString("</hunk>\n")
	}
}

func writeXMLThread(b *strings.Builder, t *github.ReviewThread, includeDiffHunk bool) {
	if len(t.Comments) == 0 {
		return
	}

	attrs := fmt.Sprintf(`line="%d"`, t.Line)
	if t.StartLine > 0 && t.StartLine != t.Line {
		attrs += fmt.Sprintf(` start_line="%d"`, t.StartLine)
	}

	if len(t.Comments) == 1 {
		c := t.Comments[0]
		if len(c.Suggestions) == 0 && (!includeDiffHunk || c.DiffHunk == "") {
			fmt.Fprintf(
				b,
				"<thread %s author=\"%s\">%s</thread>\n",
				attrs,
				c.Author,
				github.StripBloat(c.Body),
			)
			return
		}
	}

	fmt.Fprintf(b, "<thread %s>\n", attrs)
	for _, c := range t.Comments {
		fmt.Fprintf(b, "<comment author=\"%s\">%s</comment>\n", c.Author, github.StripBloat(c.Body))
		writeXMLCommentSuggestions(b, c)
	}
	if includeDiffHunk {
		if hunk := threadDiffHunk(t); hunk != "" {
			fmt.Fprintf(b, "<diff_hunk>\n%s\n</diff_hunk>\n", strings.TrimRight(hunk, "\n"))
		}
	}
	b.WriteString("</thread>\n")
}

func threadDiffHunk(t *github.ReviewThread) string {
	for _, c := range t.Comments {
		if c.DiffHunk != "" {
			return c.DiffHunk
		}
	}
	return ""
}

func writeXMLCommentSuggestions(b *strings.Builder, c github.Comment) {
	for _, s := range c.Suggestions {
		attrs := ""
		if s.Source != "" {
			attrs = fmt.Sprintf(` source=%q`, s.Source)
		}
		if s.Path != "" {
			attrs += fmt.Sprintf(` path=%q`, s.Path)
		}
		if s.StartLine > 0 {
			attrs += fmt.Sprintf(` start_line="%d"`, s.StartLine)
		}
		if s.EndLine > 0 {
			attrs += fmt.Sprintf(` end_line="%d"`, s.EndLine)
		}
		fmt.Fprintf(b, "<suggestion%s>\n%s\n</suggestion>\n", attrs, s.Code)
	}
}

// Filename returns the output filename for a review bundle.
func Filename(pr *github.PR, diff *git.BranchDiffResult) string {
	if pr != nil {
		return fmt.Sprintf("pr-%d_review.xml", pr.Number)
	}
	safeCurrent := strings.ReplaceAll(diff.CurrentBranch, "/", "-")
	safeBase := strings.ReplaceAll(diff.BaseBranch, "/", "-")
	return fmt.Sprintf("%s_%s_review.xml", safeCurrent, safeBase)
}
