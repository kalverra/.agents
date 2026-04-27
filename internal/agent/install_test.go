package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
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

	require.NoError(t, mergeHooksFromSnippet(settingsPath, snippet))

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

func TestSkillDestPlan_ListsRemovedVersusRepo(t *testing.T) {
	t.Parallel()

	destRoot := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(destRoot, "keep"), 0o750))
	require.NoError(t, os.MkdirAll(filepath.Join(destRoot, "drop"), 0o750))

	repoSkillPath := filepath.Join(t.TempDir(), "nested", "keep")
	inst := &Installer{}
	p, err := inst.skillDestPlan("test", destRoot, []string{repoSkillPath})
	require.NoError(t, err)
	require.Equal(t, "test", p.Label)
	require.Equal(t, destRoot, p.DestRoot)
	require.Equal(t, []string{"drop"}, p.Removed)
	require.Equal(t, []string{"keep"}, p.Installed)
}

func TestComputeSelection_ForcedClaude(t *testing.T) {
	t.Parallel()

	inst := &Installer{
		Targets: ParseTargets("claude"),
	}
	detected := map[Agent]bool{Claude: false}
	sel := computeSelection(inst, detected, true)
	require.True(t, sel.DidClaude)
}

func TestComputeSelection_ForcedCodex(t *testing.T) {
	t.Parallel()

	inst := &Installer{
		Targets: ParseTargets("codex"),
	}
	detected := map[Agent]bool{Codex: false}
	sel := computeSelection(inst, detected, true)
	require.True(t, sel.DidCodex)
	require.True(t, sel.anyAgent())
	require.False(t, sel.hookScriptsNeeded())
}

func TestComputeSelection_AntigravityNoStripWhenHooksKept(t *testing.T) {
	t.Parallel()

	inst := &Installer{
		WithHooks: true,
	}
	detected := map[Agent]bool{
		Gemini:      true,
		Antigravity: true,
	}
	sel := computeSelection(inst, detected, false)
	require.True(t, sel.DidGemini)
	require.True(t, sel.DidAntigravity)
	require.False(t, sel.StripGeminiMarkdown)
}

func TestCopySkills_RemovesExtraAndCopiesRepo(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	skillA := filepath.Join(repoRoot, "alpha")
	require.NoError(t, os.MkdirAll(skillA, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(skillA, "SKILL.md"), []byte("# a"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(skillA, "extra.txt"), []byte("x"), 0o600))

	dest := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dest, "orphan"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dest, "loose.txt"), []byte("y"), 0o600))

	inst := &Installer{}
	require.NoError(t, inst.copySkills([]string{skillA}, dest))

	entries, err := os.ReadDir(dest)
	require.NoError(t, err)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	require.Equal(t, []string{"alpha"}, names)

	got, err := os.ReadFile(filepath.Join(dest, "alpha", "SKILL.md")) //nolint:gosec // test temp path
	require.NoError(t, err)
	require.Equal(t, "# a", string(got))
}

func TestCopySkillsPreservingExtras_DoesNotRemoveExistingCodexSkills(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	skillA := filepath.Join(repoRoot, "alpha")
	require.NoError(t, os.MkdirAll(skillA, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(skillA, "SKILL.md"), []byte("# a"), 0o600))

	dest := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dest, ".system"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dest, ".system", "SKILL.md"), []byte("# system"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(dest, "manual"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dest, "manual", "SKILL.md"), []byte("# manual"), 0o600))

	inst := &Installer{}
	require.NoError(t, inst.copySkillsPreservingExtras([]string{skillA}, dest))

	for _, name := range []string{".system", "manual", "alpha"} {
		_, err := os.Stat(filepath.Join(dest, name, "SKILL.md"))
		require.NoError(t, err)
	}
}

func TestCopySkills_EmptyRepoClearsDestination(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dest, "gone"), 0o750))

	inst := &Installer{}
	require.NoError(t, inst.copySkills(nil, dest))

	entries, err := os.ReadDir(dest)
	require.NoError(t, err)
	require.Empty(t, entries)
}
