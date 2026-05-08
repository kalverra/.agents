package ticket

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseJiraRef_issueKey(t *testing.T) {
	t.Parallel()
	got, err := ParseJiraRef("  TASK-123  ")
	require.NoError(t, err)
	assert.Equal(t, "TASK-123", got)
}

func TestParseJiraRef_issueKey_lowercase(t *testing.T) {
	t.Parallel()
	got, err := ParseJiraRef("task-42")
	require.NoError(t, err)
	assert.Equal(t, "TASK-42", got)
}

func TestParseJiraRef_notJira_bareDigits(t *testing.T) {
	t.Parallel()
	_, err := ParseJiraRef("12345")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotJiraRef)
}

func TestParseJiraRef_notJira_todoistURL(t *testing.T) {
	t.Parallel()
	_, err := ParseJiraRef("https://app.todoist.com/app/task/abc-123")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotJiraRef)
}

func TestParseJiraRef_browseURL(t *testing.T) {
	t.Parallel()
	raw := "https://acme.atlassian.net/browse/TASK-999"
	got, err := ParseJiraRef(raw)
	require.NoError(t, err)
	assert.Equal(t, "TASK-999", got)
}

func TestParseJiraRef_browseURL_trailingPath(t *testing.T) {
	t.Parallel()
	raw := "https://acme.atlassian.net/browse/FOO-1/overview"
	got, err := ParseJiraRef(raw)
	require.NoError(t, err)
	assert.Equal(t, "FOO-1", got)
}

func TestParseJiraRef_selectedIssueQuery(t *testing.T) {
	t.Parallel()
	raw := "https://acme.atlassian.net/jira/software/c/projects/PRO/boards/1?selectedIssue=PRO-7"
	got, err := ParseJiraRef(raw)
	require.NoError(t, err)
	assert.Equal(t, "PRO-7", got)
}

func TestParseJiraRef_issuesPathSegment(t *testing.T) {
	t.Parallel()
	raw := "https://acme.atlassian.net/jira/software/projects/PRO/issues/PRO-12"
	got, err := ParseJiraRef(raw)
	require.NoError(t, err)
	assert.Equal(t, "PRO-12", got)
}

func TestParseJiraRef_empty(t *testing.T) {
	t.Parallel()
	_, err := ParseJiraRef("   ")
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrNotJiraRef)
}

func TestParseJiraRef_atlassianHostNoKey(t *testing.T) {
	t.Parallel()
	_, err := ParseJiraRef("https://acme.atlassian.net/jira/software/projects/PRO/boards/1")
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrNotJiraRef)
}
