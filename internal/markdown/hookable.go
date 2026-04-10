// Package markdown provides utilities for processing agent instruction markdown.
package markdown

import (
	"regexp"
	"strings"
)

var sectionRe = regexp.MustCompile(
	`(?m)^<hookable name="\w+">\s*\n` +
		`(?:.*?\n)*?` +
		`</hookable>\s*\n`,
)

var openTagRe = regexp.MustCompile(`^<hookable name="\w+">$`)
var closeTagRe = regexp.MustCompile(`^</hookable>$`)

// StripHookableSections removes entire <hookable name="...">...</hookable> blocks.
// Used when hooks are installed (the hookable regions are handled by hooks instead).
func StripHookableSections(text string) string {
	return sectionRe.ReplaceAllString(text, "")
}

// StripHookableDelimiterLines removes only the <hookable> and </hookable> tag lines,
// keeping the inner content. Used when deploying without hooks.
func StripHookableDelimiterLines(text string) string {
	var out strings.Builder
	for line := range strings.SplitSeq(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if openTagRe.MatchString(trimmed) || closeTagRe.MatchString(trimmed) {
			continue
		}
		out.WriteString(line)
		out.WriteString("\n")
	}
	// The original text may not end with \n; trim the extra one we appended.
	result := out.String()
	if len(result) > 0 && (len(text) == 0 || text[len(text)-1] != '\n') {
		result = result[:len(result)-1]
	}
	return result
}
