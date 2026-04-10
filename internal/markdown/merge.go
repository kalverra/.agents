package markdown

import (
	"os"
	"regexp"
	"strings"
)

const userAgentsPlaceholder = "<!-- Instructions from USER_AGENTS.md are appended here during install -->"

var userBlockRe = regexp.MustCompile(`(?s)(<user>).*?(</user>)`)

// MergeUserAgents inserts the content of userSrcPath into the template body.
//
// Injection order:
//  1. Replace the install-time HTML comment placeholder.
//  2. Replace the inner content of <user>...</user>.
//  3. Append to the end of the file.
func MergeUserAgents(body, userSrcPath string) (string, error) {
	data, err := os.ReadFile(userSrcPath) //nolint:gosec // userSrcPath is a user-controlled config path
	if err != nil {
		if os.IsNotExist(err) {
			return body, nil
		}
		return "", err
	}

	userContent := strings.TrimSpace(string(data))
	if userContent == "" {
		return body, nil
	}

	if strings.Contains(body, userAgentsPlaceholder) {
		return strings.Replace(body, userAgentsPlaceholder, userContent, 1), nil
	}

	if userBlockRe.MatchString(body) {
		return userBlockRe.ReplaceAllStringFunc(body, func(match string) string {
			parts := userBlockRe.FindStringSubmatch(match)
			return parts[1] + "\n" + userContent + "\n" + parts[2]
		}), nil
	}

	return strings.TrimRight(body, "\n") + "\n\n" + userContent + "\n", nil
}
