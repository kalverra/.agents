package ticket

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kalverra/agents/internal/git"
)

func TestIsWorkRepo(t *testing.T) {
	t.Parallel()

	workRepos := []string{
		"acme/*",
		"github.com/work-org/*",
		"special/repo-x",
		"github.com/corp/portal",
	}

	tests := []struct {
		name     string
		repo     *git.RepoInfo
		expected bool
	}{
		{
			name:     "wildcard owner match acme",
			repo:     &git.RepoInfo{Owner: "acme", Name: "service-a", Branch: "main"},
			expected: true,
		},
		{
			name:     "wildcard github owner match work-org",
			repo:     &git.RepoInfo{Owner: "work-org", Name: "api", Branch: "feature/PROJ-1"},
			expected: true,
		},
		{
			name:     "exact match special/repo-x",
			repo:     &git.RepoInfo{Owner: "special", Name: "repo-x", Branch: "main"},
			expected: true,
		},
		{
			name:     "exact github match corp/portal",
			repo:     &git.RepoInfo{Owner: "corp", Name: "portal", Branch: "main"},
			expected: true,
		},
		{
			name:     "personal repo not matched",
			repo:     &git.RepoInfo{Owner: "kalverra", Name: "dotfiles", Branch: "main"},
			expected: false,
		},
		{
			name:     "nil repo info",
			repo:     nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsWorkRepo(tt.repo, workRepos)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestExtractJiraKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		branch   string
		expected string
	}{
		{"feature/PROJ-1234-add-widget", "PROJ-1234"},
		{"PROJ-99/bugfix", "PROJ-99"},
		{"bugfix/DX-500", "DX-500"},
		{"add-widget/DX-123", "DX-123"},
		{"adam/DX-100", "DX-100"},
		{"PROJ-123", "PROJ-123"},
		{"main", ""},
		{"master", ""},
		{"dev", ""},
		{"feature/no-jira-key-here", ""},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			t.Parallel()
			got := ExtractJiraKey(tt.branch)
			assert.Equal(t, tt.expected, got)
		})
	}
}
