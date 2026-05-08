package ticket

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// ErrNotJiraRef means the input is not a Jira issue id or URL (caller may try Todoist).
var ErrNotJiraRef = errors.New("not a Jira issue reference")

// jiraIssueKey matches Jira issue keys after uppercasing (e.g. TASK-123).
var jiraIssueKey = regexp.MustCompile(`^[A-Z][A-Z0-9_]*-\d+$`)

// jiraIssuesPathKey captures .../issues/KEY (Jira Software URLs).
var jiraIssuesPathKey = regexp.MustCompile(`(?i)/issues/([A-Z][A-Z0-9_]*-\d+)(?:/|$)`)

// ParseJiraRef returns a Jira issue key from a bare key (e.g. TASK-123) or a Jira issue URL.
// If the value is not Jira-shaped, it returns ("", ErrNotJiraRef).
// Malformed URLs or recognized Jira hosts without a resolvable key return a different error.
func ParseJiraRef(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("issue id or URL is required")
	}
	if !strings.Contains(s, "://") {
		if k, ok := normalizeIssueKey(s); ok {
			return k, nil
		}
		return "", ErrNotJiraRef
	}

	u, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("parsing URL: %w", err)
	}

	host := strings.ToLower(strings.TrimPrefix(u.Hostname(), "www."))
	if host == "app.todoist.com" || host == "todoist.com" {
		return "", ErrNotJiraRef
	}

	if q := strings.TrimSpace(u.Query().Get("selectedIssue")); q != "" {
		if k, ok := normalizeIssueKey(q); ok {
			return k, nil
		}
	}

	if k := keyFromBrowsePath(u.Path); k != "" {
		return k, nil
	}

	if m := jiraIssuesPathKey.FindStringSubmatch(u.Path); len(m) == 2 {
		if k, ok := normalizeIssueKey(m[1]); ok {
			return k, nil
		}
	}

	if isLikelyJiraHost(host) {
		return "", fmt.Errorf("jira URL %q does not contain a resolvable issue key", raw)
	}
	return "", ErrNotJiraRef
}

func normalizeIssueKey(s string) (string, bool) {
	k := strings.ToUpper(strings.TrimSpace(s))
	if jiraIssueKey.MatchString(k) {
		return k, true
	}
	return "", false
}

func keyFromBrowsePath(path string) string {
	lower := strings.ToLower(path)
	idx := strings.Index(lower, "/browse/")
	if idx < 0 {
		return ""
	}
	rest := strings.Trim(path[idx+len("/browse/"):], "/")
	if rest == "" {
		return ""
	}
	if slash := strings.Index(rest, "/"); slash >= 0 {
		rest = rest[:slash]
	}
	seg, err := url.PathUnescape(rest)
	if err != nil {
		seg = rest
	}
	if k, ok := normalizeIssueKey(seg); ok {
		return k
	}
	return ""
}

func isLikelyJiraHost(host string) bool {
	if host == "" {
		return false
	}
	if strings.HasSuffix(host, ".atlassian.net") {
		return true
	}
	return strings.Contains(host, "jira.")
}
