package hooks

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	// internal/hooks -> repo root is two levels up
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func runPrependHook(t *testing.T, agentType, stdin string) []byte {
	t.Helper()
	hook := filepath.Join(repoRoot(t), "hooks", "rtk-prepend.sh")
	//nolint:gosec // G204: hook path is repoRoot + fixed "hooks/rtk-prepend.sh", not user input
	cmd := exec.Command(
		"bash",
		hook,
	)
	cmd.Env = append(os.Environ(), "AGENT_TYPE="+agentType)
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.Output()
	require.NoError(t, err)
	return out
}

func TestRtkPrependHook_RewrittenCommandSetsSuppressHookWarningEnv(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name   string
		agent  string
		stdin  string
		jsonFn func(t *testing.T, raw []byte) string
	}{
		{
			name:  "cursor",
			agent: "cursor",
			stdin: `{"arguments":{"command":"echo x"}}`,
			jsonFn: func(t *testing.T, raw []byte) string {
				t.Helper()
				var v struct {
					UpdatedInput struct {
						Command string `json:"command"`
					} `json:"updated_input"`
				}
				require.NoError(t, json.Unmarshal(raw, &v))
				return v.UpdatedInput.Command
			},
		},
		{
			name:  "claude",
			agent: "claude",
			stdin: `{"tool_input":{"command":"echo x"}}`,
			jsonFn: func(t *testing.T, raw []byte) string {
				t.Helper()
				var v struct {
					HookSpecificOutput struct {
						UpdatedInput struct {
							Command string `json:"command"`
						} `json:"updatedInput"`
					} `json:"hookSpecificOutput"`
				}
				require.NoError(t, json.Unmarshal(raw, &v))
				return v.HookSpecificOutput.UpdatedInput.Command
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := runPrependHook(t, tc.agent, tc.stdin)
			rewritten := tc.jsonFn(t, out)
			assert.Contains(t, rewritten, "RTK_SUPPRESS_HOOK_WARNING=1",
				"rewritten command should ask rtk to suppress hook warnings (see rtk-ai/rtk#682)")
		})
	}
}

func TestRtkPrependHook_FilteredCommandOmitsHookWarning(t *testing.T) {
	t.Parallel()

	out := runPrependHook(t, "cursor", `{"arguments":{"command":"echo marker"}}`)
	var payload struct {
		UpdatedInput struct {
			Command string `json:"command"`
		} `json:"updated_input"`
	}
	require.NoError(t, json.Unmarshal(out, &payload))
	rewritten := payload.UpdatedInput.Command
	require.NotEmpty(t, rewritten)

	tmp := t.TempDir()
	fakeRtk := filepath.Join(tmp, "rtk")
	script := "#!/bin/sh\n" +
		"printf '%s\\n' '[rtk] /!\\ No hook installed — run `rtk init -g` for automatic token savings' >&2\n" +
		"exec \"$@\"\n"
	//nolint:gosec // G204: fakeRtk is a temporary file in the test's temp dir, not arbitrary user input
	require.NoError(t, os.WriteFile(fakeRtk, []byte(script), 0o700))

	//nolint:gosec // G204: rewritten is hook output from this test's stdin, not arbitrary user input
	runCmd := exec.Command(
		"bash",
		"-c",
		rewritten,
	)
	runCmd.Env = append(os.Environ(), "PATH="+tmp+string(os.PathListSeparator)+os.Getenv("PATH"))
	combined, err := runCmd.CombinedOutput()
	require.NoError(t, err, "combined: %s", combined)

	outStr := string(combined)
	assert.NotContains(t, outStr, "No hook installed", "filter should drop rtk hook nag line")
	assert.Contains(t, outStr, "marker", "child command should still run")
}
