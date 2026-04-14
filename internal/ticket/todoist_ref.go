package ticket

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// todoistLongID matches Todoist-style task ids (suffix in /app/task/… URLs) and long numeric ids.
var todoistLongID = regexp.MustCompile(`[A-Za-z0-9]{8,}`)

// ParseTaskRef returns a REST API task id from a bare id or a Todoist task URL.
// Supports:
//   - Plain task id (unchanged)
//   - https://app.todoist.com/app/task/<slug>-<id> (id is the last long alphanumeric run)
//   - https://app.todoist.com/app/task/<id>
//   - https://todoist.com/showTask?id=<id>
func ParseTaskRef(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("task id or URL is required")
	}
	if !strings.Contains(s, "://") {
		return s, nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("parsing URL: %w", err)
	}

	host := strings.ToLower(strings.TrimPrefix(u.Hostname(), "www."))
	if host != "app.todoist.com" && host != "todoist.com" {
		return "", fmt.Errorf("unsupported host %q (expected app.todoist.com or todoist.com)", u.Hostname())
	}

	lowerPath := strings.ToLower(u.Path)
	if strings.Contains(lowerPath, "showtask") {
		id := strings.TrimSpace(u.Query().Get("id"))
		if id == "" {
			return "", fmt.Errorf("todoist showTask URL has no id= query parameter")
		}
		return id, nil
	}

	idx := strings.Index(lowerPath, "/task/")
	if idx < 0 {
		return "", fmt.Errorf("todoist URL has no /task/ segment")
	}

	rest := strings.Trim(u.Path[idx+len("/task/"):], "/")
	if rest == "" {
		return "", fmt.Errorf("todoist task URL is missing the task segment")
	}
	first := rest
	if slash := strings.Index(first, "/"); slash >= 0 {
		first = first[:slash]
	}
	seg, err := url.PathUnescape(first)
	if err != nil {
		return "", fmt.Errorf("decoding task path: %w", err)
	}
	return extractTaskIDFromSegment(seg), nil
}

func extractTaskIDFromSegment(seg string) string {
	if seg == "" {
		return ""
	}
	matches := todoistLongID.FindAllString(seg, -1)
	if len(matches) == 0 {
		return seg
	}
	return matches[len(matches)-1]
}
