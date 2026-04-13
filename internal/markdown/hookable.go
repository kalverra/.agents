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
	lines := strings.Split(text, "\n")
	var filtered []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if openTagRe.MatchString(trimmed) || closeTagRe.MatchString(trimmed) {
			continue
		}
		filtered = append(filtered, line)
	}
	return strings.Join(filtered, "\n")
}
