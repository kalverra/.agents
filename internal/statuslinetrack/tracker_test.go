package statuslinetrack

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSync_TracksStartTimeAcrossCalls(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	session := "sess-1"
	start := time.Date(2026, 6, 8, 14, 0, 0, 0, time.UTC)
	later := start.Add(2*time.Minute + 15*time.Second)

	active := []ActiveTask{
		{Key: "0", Name: "go test ./...", Sort: 0},
		{Key: "1", Name: "npm run build", Sort: 1},
	}

	first, err := Sync(root, session, active, start)
	require.NoError(t, err)
	require.Len(t, first, 2)
	require.Equal(t, "go test ./...", first[0].Name)
	require.Equal(t, start, first[0].StartedAt)

	second, err := Sync(root, session, active, later)
	require.NoError(t, err)
	require.Equal(t, start, second[0].StartedAt)
	require.Equal(t, start, second[1].StartedAt)
	require.Equal(t, later.Sub(start), later.Sub(second[0].StartedAt))
}

func TestSync_DropsFinishedTasks(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	session := "sess-2"
	now := time.Date(2026, 6, 8, 14, 0, 0, 0, time.UTC)

	_, err := Sync(root, session, []ActiveTask{
		{Key: "0", Name: "go test ./...", Sort: 0},
		{Key: "1", Name: "npm run build", Sort: 1},
	}, now)
	require.NoError(t, err)

	remaining, err := Sync(root, session, []ActiveTask{
		{Key: "1", Name: "npm run build", Sort: 1},
	}, now.Add(time.Minute))
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	require.Equal(t, "1", remaining[0].Key)
}
