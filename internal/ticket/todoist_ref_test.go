package ticket

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTaskRef_plainID(t *testing.T) {
	t.Parallel()
	got, err := ParseTaskRef("  12345  ")
	require.NoError(t, err)
	assert.Equal(t, "12345", got)
}

func TestParseTaskRef_appTaskURL_withSlug(t *testing.T) {
	t.Parallel()
	raw := "https://app.todoist.com/app/task/setup-vector-database-6g9Ww74qF8V7Q4Rw"
	got, err := ParseTaskRef(raw)
	require.NoError(t, err)
	assert.Equal(t, "6g9Ww74qF8V7Q4Rw", got)
}

func TestParseTaskRef_appTaskURL_idOnly(t *testing.T) {
	t.Parallel()
	raw := "https://app.todoist.com/app/task/6g9Ww74qF8V7Q4Rw"
	got, err := ParseTaskRef(raw)
	require.NoError(t, err)
	assert.Equal(t, "6g9Ww74qF8V7Q4Rw", got)
}

func TestParseTaskRef_todoistComHost(t *testing.T) {
	t.Parallel()
	raw := "https://todoist.com/app/task/my-task-6g9Ww74qF8V7Q4Rw"
	got, err := ParseTaskRef(raw)
	require.NoError(t, err)
	assert.Equal(t, "6g9Ww74qF8V7Q4Rw", got)
}

func TestParseTaskRef_showTask(t *testing.T) {
	t.Parallel()
	raw := "https://todoist.com/showTask?id=999888777"
	got, err := ParseTaskRef(raw)
	require.NoError(t, err)
	assert.Equal(t, "999888777", got)
}

func TestParseTaskRef_showTask_missingID(t *testing.T) {
	t.Parallel()
	_, err := ParseTaskRef("https://todoist.com/showTask")
	require.Error(t, err)
}

func TestParseTaskRef_badHost(t *testing.T) {
	t.Parallel()
	_, err := ParseTaskRef("https://example.com/app/task/abc")
	require.Error(t, err)
}

func TestParseTaskRef_noTaskSegment(t *testing.T) {
	t.Parallel()
	_, err := ParseTaskRef("https://app.todoist.com/app/inbox")
	require.Error(t, err)
}

func TestParseTaskRef_trailingSlash(t *testing.T) {
	t.Parallel()
	raw := "https://app.todoist.com/app/task/setup-vector-database-6g9Ww74qF8V7Q4Rw/"
	got, err := ParseTaskRef(raw)
	require.NoError(t, err)
	assert.Equal(t, "6g9Ww74qF8V7Q4Rw", got)
}
