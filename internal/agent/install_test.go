package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeHooksFromSnippet_CursorSnippetIsFlatPreToolUse(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	settingsPath := filepath.Join(tmp, "hooks.json")

	snippet := map[string]any{
		"hooks": map[string]any{
			"preToolUse": []any{
				map[string]any{
					"type":    "command",
					"command": "/fake/hooks/cursor-rtk.sh",
					"matcher": "run_shell_command",
				},
			},
		},
	}

	require.NoError(t, mergeHooksFromSnippet(settingsPath, snippet, false))

	//nolint:gosec // G304: path is under t.TempDir(), not user-controlled input
	data, err := os.ReadFile(settingsPath)
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(data, &out))

	ver, ok := out["version"].(float64)
	require.True(t, ok, "Cursor hooks.json should include numeric version")
	require.InDelta(t, 1.0, ver, 0)

	hooksSection, ok := out["hooks"].(map[string]any)
	require.True(t, ok)
	arr, ok := hooksSection["preToolUse"].([]any)
	require.True(t, ok)
	require.Len(t, arr, 1)

	entry, ok := arr[0].(map[string]any)
	require.True(t, ok)

	_, hasNestedHooks := entry["hooks"]
	require.False(t, hasNestedHooks, "Cursor hook entries must not use nested hooks array")

	require.Equal(t, "/fake/hooks/cursor-rtk.sh", entry["command"])
	require.Equal(t, "run_shell_command", entry["matcher"])
	require.Equal(t, "command", entry["type"])
}

func TestCursorHooksSnippetOnDiskIsFlat(t *testing.T) {
	t.Parallel()

	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	path := filepath.Join(repoRoot, "hooks", "cursor-hooks-snippet.json")
	//nolint:gosec // G304: fixed repo path from runtime.Caller, not user input
	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	var snippet map[string]any
	require.NoError(t, json.Unmarshal(raw, &snippet))

	hooksSection, ok := snippet["hooks"].(map[string]any)
	require.True(t, ok)
	arr, ok := hooksSection["preToolUse"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, arr)

	for i, item := range arr {
		entry, ok := item.(map[string]any)
		require.True(t, ok, "preToolUse[%d]", i)
		_, nested := entry["hooks"]
		require.False(
			t,
			nested,
			"cursor-hooks-snippet.json must use Cursor flat hook objects, not nested hooks[] (see create-hook skill)",
		)
		require.NotEmpty(t, entry["command"])
	}
}
