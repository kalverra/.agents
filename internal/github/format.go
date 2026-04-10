package github

import (
	"fmt"
	"strings"
)

// FormatPR renders a PR and its review activity as readable markdown.
func FormatPR(pr *PR, includeResolved bool) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# PR #%d: %s\n\n", pr.Number, pr.Title)
	fmt.Fprintf(&b, "- **URL:** %s\n", pr.URL)
	fmt.Fprintf(&b, "- **Branch:** %s -> %s\n", pr.HeadRef, pr.BaseRef)
	fmt.Fprintf(&b, "- **Author:** %s\n", pr.Author)
	if pr.ReviewDecision != "" {
		fmt.Fprintf(&b, "- **Review Decision:** %s\n", pr.ReviewDecision)
	}

	b.WriteString("\n## PR Description\n\n")
	if pr.Body != "" {
		b.WriteString(pr.Body)
		b.WriteString("\n")
	} else {
		b.WriteString("*(no description)*\n")
	}

	formatReviews(&b, pr.Reviews)

	var unresolved, resolved []ReviewThread
	for _, t := range pr.Threads {
		if t.IsResolved {
			resolved = append(resolved, t)
		} else {
			unresolved = append(unresolved, t)
		}
	}

	formatUnresolvedThreads(&b, unresolved)
	formatGeneralComments(&b, pr.Comments)
	formatResolvedThreads(&b, resolved, includeResolved)

	return b.String()
}

func formatReviews(b *strings.Builder, reviews []Review) {
	if len(reviews) == 0 {
		return
	}
	b.WriteString("\n## Reviews\n\n")
	for _, r := range reviews {
		date := r.CreatedAt.Format("2006-01-02")
		fmt.Fprintf(b, "### @%s -- %s (%s)\n", r.Author, r.State, date)
		if r.Body != "" {
			fmt.Fprintf(b, "%s\n", r.Body)
		}
		b.WriteString("\n")
	}
}

func formatUnresolvedThreads(b *strings.Builder, threads []ReviewThread) {
	fmt.Fprintf(b, "\n## Unresolved Review Threads (%d)\n\n", len(threads))
	if len(threads) == 0 {
		b.WriteString("No unresolved threads.\n")
		return
	}
	for i, t := range threads {
		formatThread(b, t)
		if i < len(threads)-1 {
			b.WriteString("\n---\n\n")
		}
	}
}

func formatGeneralComments(b *strings.Builder, comments []Comment) {
	if len(comments) == 0 {
		return
	}
	b.WriteString("\n## General Comments\n\n")
	for _, c := range comments {
		date := c.CreatedAt.Format("2006-01-02")
		fmt.Fprintf(b, "**@%s** (%s):\n", c.Author, date)
		writeQuoted(b, c.Body)
		b.WriteString("\n")
	}
}

func formatResolvedThreads(b *strings.Builder, threads []ReviewThread, full bool) {
	if len(threads) == 0 {
		return
	}
	fmt.Fprintf(b, "\n## Resolved Threads (%d)\n\n", len(threads))
	if !full {
		b.WriteString("*(pass --include-resolved to expand)*\n")
		return
	}
	for i, t := range threads {
		formatThread(b, t)
		if i < len(threads)-1 {
			b.WriteString("\n---\n\n")
		}
	}
}

func formatThread(b *strings.Builder, t ReviewThread) {
	loc := t.Path
	if t.Line > 0 {
		loc = fmt.Sprintf("%s:%d", t.Path, t.Line)
	}
	label := "Thread"
	if t.IsOutdated {
		label = "Thread (outdated)"
	}
	fmt.Fprintf(b, "### %s: %s\n", label, loc)
	for _, c := range t.Comments {
		date := c.CreatedAt.Format("2006-01-02")
		fmt.Fprintf(b, "**@%s** (%s):\n", c.Author, date)
		writeQuoted(b, c.Body)
		b.WriteString("\n")
	}
}

func writeQuoted(b *strings.Builder, body string) {
	for line := range strings.SplitSeq(body, "\n") {
		fmt.Fprintf(b, "> %s\n", line)
	}
}
