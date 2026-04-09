package cmd

import (
	"regexp"
	"strings"
)

var hookableSectionRe = regexp.MustCompile(`(?m)^<hookable name="\w+">\s*\n(?:.*?\n)*?</hookable>\s*\n`)
var hookableStartRe = regexp.MustCompile(`^<hookable name="\w+">$`)
var hookableEndRe = regexp.MustCompile(`^</hookable>$`)

func stripHookableSections(text string) string {
	return hookableSectionRe.ReplaceAllString(text, "")
}

func stripHookableDelimiterLines(text string) string {
	var out []string
	// Need to handle different newline variations
	lines := strings.SplitAfter(text, "\n")
	for _, line := range lines {
		s := strings.TrimSpace(line)
		if hookableStartRe.MatchString(s) {
			continue
		}
		if hookableEndRe.MatchString(s) {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "")
}
