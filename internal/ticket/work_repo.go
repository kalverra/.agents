package ticket

import (
	"regexp"
	"strings"

	"github.com/kalverra/agents/internal/git"
)

var jiraKeyRegex = regexp.MustCompile(`\b([A-Z][A-Z0-9_]+-[0-9]+)\b`)

// ExtractJiraKey finds the first Jira issue key (e.g. PROJ-123) in a branch name.
func ExtractJiraKey(branch string) string {
	branch = strings.TrimSpace(branch)
	if branch == "" || branch == "main" || branch == "master" || branch == "dev" || branch == "develop" {
		return ""
	}
	m := jiraKeyRegex.FindStringSubmatch(branch)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

// IsWorkRepo returns true if the repo matches any pattern in workRepos.
// Supported pattern formats:
// - "owner/*" or "github.com/owner/*" (wildcard for org/owner)
// - "owner/repo" or "github.com/owner/repo" (exact match)
func IsWorkRepo(repo *git.RepoInfo, workRepos []string) bool {
	if repo == nil || repo.Owner == "" {
		return false
	}

	repoOwner := strings.ToLower(strings.TrimSpace(repo.Owner))
	repoName := strings.ToLower(strings.TrimSpace(repo.Name))
	fullPath := repoOwner + "/" + repoName

	for _, pattern := range workRepos {
		p := strings.ToLower(strings.TrimSpace(pattern))
		if p == "" {
			continue
		}
		// Strip leading protocol / domain prefixes like github.com/
		p = strings.TrimPrefix(p, "https://")
		p = strings.TrimPrefix(p, "http://")
		if idx := strings.Index(p, "/"); idx != -1 && strings.Contains(p[:idx], ".") {
			p = p[idx+1:]
		}

		if before, ok := strings.CutSuffix(p, "/*"); ok {
			org := before
			if repoOwner == org {
				return true
			}
		} else if p == fullPath || p == repoName {
			return true
		}
	}

	return false
}
