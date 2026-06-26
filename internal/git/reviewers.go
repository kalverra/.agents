package git

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var hunkHeaderRe = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+\d+(?:,\d+)? @@`)

// SuggestReviewers analyzes the modified files in the BranchDiffResult to determine
// the authors who wrote the lines being changed or deleted. It returns up to `limit`
// most frequent authors, excluding `excludeAuthor`.
func SuggestReviewers(dir string, diffResult *BranchDiffResult, excludeAuthor string, limit int) ([]string, error) {
	authorCounts := make(map[string]int)

	for _, f := range diffResult.Files {
		if f.Patch == "" || f.Status == "added" || f.Status == "deleted" || f.Status == "binary" {
			continue
		}

		blamedLines := extractBlameLines(f.Patch)
		if len(blamedLines) == 0 {
			continue
		}

		authorsByLine, err := runBlame(dir, diffResult.MergeBase, f.Path)
		if err != nil {
			continue
		}

		for _, lineNum := range blamedLines {
			author := authorsByLine[lineNum]
			if author != "" && author != excludeAuthor && author != "Not Committed Yet" {
				authorCounts[author]++
			}
		}
	}

	type authorCount struct {
		name  string
		count int
	}
	var acs []authorCount
	for name, count := range authorCounts {
		acs = append(acs, authorCount{name, count})
	}

	sort.Slice(acs, func(i, j int) bool {
		if acs[i].count == acs[j].count {
			return acs[i].name < acs[j].name
		}
		return acs[i].count > acs[j].count
	})

	var result []string
	for i := 0; i < len(acs) && i < limit; i++ {
		result = append(result, acs[i].name)
	}

	return result, nil
}

func extractBlameLines(patch string) []int {
	var lines []int
	var currentLine int
	inHunk := false
	var hunkAddedLines int
	var hunkDeletedLines int
	var hunkStartLine int

	for line := range strings.SplitSeq(patch, "\n") {
		if m := hunkHeaderRe.FindStringSubmatch(line); m != nil {
			// If previous hunk was pure addition, blame the context line before it
			if inHunk && hunkDeletedLines == 0 && hunkAddedLines > 0 {
				if hunkStartLine > 0 {
					lines = append(lines, hunkStartLine)
				}
			}

			currentLine, _ = strconv.Atoi(m[1])
			hunkStartLine = currentLine
			inHunk = true
			hunkAddedLines = 0
			hunkDeletedLines = 0
			continue
		}
		if !inHunk || len(line) == 0 {
			continue
		}

		char := line[0]
		switch char {
		case ' ':
			currentLine++
		case '-':
			lines = append(lines, currentLine)
			hunkDeletedLines++
			currentLine++
		case '+':
			hunkAddedLines++
		}
	}

	if inHunk && hunkDeletedLines == 0 && hunkAddedLines > 0 {
		if hunkStartLine > 0 {
			lines = append(lines, hunkStartLine)
		}
	}

	return lines
}

func runBlame(dir, mergeBase, file string) (map[int]string, error) {
	out, err := gitOutput(dir, "blame", "--line-porcelain", mergeBase, "--", file)
	if err != nil {
		return nil, err
	}

	authorsByLine := make(map[int]string)
	var currentLine int

	for line := range strings.SplitSeq(out, "\n") {
		// A line starting with 40-char hash is the header for a block
		if len(line) >= 40 && strings.IndexByte(line, ' ') == 40 {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				currentLine, _ = strconv.Atoi(parts[2])
			}
			continue
		}
		if after, ok := strings.CutPrefix(line, "author "); ok {
			author := after
			authorsByLine[currentLine] = author
		}
	}
	return authorsByLine, nil
}

// GetCurrentAuthor returns the current user.name from git config.
func GetCurrentAuthor(dir string) string {
	name, err := gitOutput(dir, "config", "user.name")
	if err != nil {
		return ""
	}
	return name
}
