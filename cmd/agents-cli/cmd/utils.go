package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var userAgentsPlaceholder = "<!-- Instructions from USER_AGENTS.md are appended here during install -->"
var userBlockRe = regexp.MustCompile(`(?s)(<user>).*?(</user>)`)

func mergeUserAgents(body string, userSrc string) string {
	b, err := os.ReadFile(userSrc)
	if err != nil {
		return body
	}
	userContent := strings.TrimSpace(string(b))
	if userContent == "" {
		return body
	}

	if strings.Contains(body, userAgentsPlaceholder) {
		return strings.Replace(body, userAgentsPlaceholder, userContent, 1)
	}

	if userBlockRe.MatchString(body) {
		return userBlockRe.ReplaceAllStringFunc(body, func(m string) string {
			matches := userBlockRe.FindStringSubmatch(m)
			return matches[1] + "\n" + userContent + "\n" + matches[2]
		})
	}

	return strings.TrimRight(body, " \t\n\r") + "\n\n" + userContent + "\n"
}

// Common repo paths and tools
func repoRoot() string {
	// For CLI, assume we are either running in repo root or binary is nearby.
	// We'll use current working directory or traverse up.
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "GLOBAL_AGENTS.md")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "."
}

func getHomeDir() string {
	h, _ := os.UserHomeDir()
	return h
}
