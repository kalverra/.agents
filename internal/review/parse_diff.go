package review

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/kalverra/agents/internal/github"
)

var hunkHeaderRegex = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

// injectComments returns the interleaved diff text, and a list of threads that were not interleaved.
func injectComments(diffText string, threads []github.ReviewThread) (string, []github.ReviewThread) {
	if len(threads) == 0 {
		return diffText, nil
	}

	// build map: file -> line -> thread
	threadMap := make(map[string]map[int][]*github.ReviewThread)
	var notInterleaved []github.ReviewThread

	for i := range threads {
		t := &threads[i]
		if t.Line == 0 {
			notInterleaved = append(notInterleaved, *t)
			continue
		}
		if threadMap[t.Path] == nil {
			threadMap[t.Path] = make(map[int][]*github.ReviewThread)
		}
		threadMap[t.Path][t.Line] = append(threadMap[t.Path][t.Line], t)
	}

	var b strings.Builder
	var currentFile string
	var currentLine int
	inHunk := false

	// Helper to collect interleaved threads
	var interleavedThreads []*github.ReviewThread

	lines := strings.SplitSeq(diffText, "\n")
	for line := range lines {
		if strings.HasPrefix(line, "[MODIFIED] ") || strings.HasPrefix(line, "[ADDED] ") ||
			strings.HasPrefix(line, "[DELETED] ") {
			parts := strings.Split(line, " ")
			if len(parts) >= 2 {
				currentFile = parts[1]
			}
			inHunk = false
			b.WriteString(line)
			b.WriteByte('\n')
			continue
		}

		if m := hunkHeaderRegex.FindStringSubmatch(line); m != nil {
			currentLine, _ = strconv.Atoi(m[1])
			inHunk = true
			b.WriteString(line)
			b.WriteByte('\n')
			continue
		}

		b.WriteString(line)
		b.WriteByte('\n')

		if inHunk {
			if len(line) > 0 {
				char := line[0]
				switch char {
				case ' ', '+':
					if threadsForLine, ok := threadMap[currentFile][currentLine]; ok {
						for _, t := range threadsForLine {
							b.WriteString(formatInterleavedThread(*t))
							interleavedThreads = append(interleavedThreads, t)
						}
					}
					currentLine++
				case '-':
					// no advance
				case '\\':
					// no newline message
				}
			} else {
				if threadsForLine, ok := threadMap[currentFile][currentLine]; ok {
					for _, t := range threadsForLine {
						b.WriteString(formatInterleavedThread(*t))
						interleavedThreads = append(interleavedThreads, t)
					}
				}
				currentLine++
			}
		}
	}

	// Figure out which threads were NOT interleaved
	interleavedMap := make(map[*github.ReviewThread]bool)
	for _, t := range interleavedThreads {
		interleavedMap[t] = true
	}

	for i := range threads {
		t := &threads[i]
		if t.Line != 0 && !interleavedMap[t] {
			notInterleaved = append(notInterleaved, *t)
		}
	}

	return strings.TrimRight(b.String(), "\n"), notInterleaved
}

func formatInterleavedThread(t github.ReviewThread) string {
	var b strings.Builder
	label := "Thread"
	if t.IsOutdated {
		label = "Thread (outdated)"
	}
	fmt.Fprintf(&b, "╭── %s\n", label)
	for _, c := range t.Comments {
		date := c.CreatedAt.Format("2006-01-02")
		fmt.Fprintf(&b, "│ **@%s** (%s):\n", c.Author, date)
		for line := range strings.SplitSeq(github.StripBloat(c.Body), "\n") {
			fmt.Fprintf(&b, "│ > %s\n", line)
		}
		b.WriteString("│\n")
	}
	b.WriteString("╰────────────────────────────────────\n")
	return b.String()
}
