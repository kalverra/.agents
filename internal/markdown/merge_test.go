package markdown_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kalverra/agents/internal/markdown"
)

func TestMergeUserAgents(t *testing.T) {
	t.Parallel()

	t.Run("missing file returns body unchanged", func(t *testing.T) {
		t.Parallel()
		got, err := markdown.MergeUserAgents("hello", "/nonexistent/USER_AGENTS.md")
		require.NoError(t, err)
		assert.Equal(t, "hello", got)
	})

	t.Run("empty file returns body unchanged", func(t *testing.T) {
		t.Parallel()
		tmp := filepath.Join(t.TempDir(), "USER_AGENTS.md")
		require.NoError(t, os.WriteFile(tmp, []byte("  \n"), 0o644)) //nolint:gosec // test helper writing temp file
		got, err := markdown.MergeUserAgents("hello", tmp)
		require.NoError(t, err)
		assert.Equal(t, "hello", got)
	})

	t.Run("replaces placeholder comment", func(t *testing.T) {
		t.Parallel()
		tmp := filepath.Join(t.TempDir(), "USER_AGENTS.md")
		require.NoError(
			t,
			os.WriteFile(tmp, []byte("user stuff"), 0o644), //nolint:gosec // test helper writing temp file
		)

		body := "before\n<!-- Instructions from USER_AGENTS.md are appended here during install -->\nafter"
		got, err := markdown.MergeUserAgents(body, tmp)
		require.NoError(t, err)
		assert.Equal(t, "before\nuser stuff\nafter", got)
	})

	t.Run("replaces user block inner content", func(t *testing.T) {
		t.Parallel()
		tmp := filepath.Join(t.TempDir(), "USER_AGENTS.md")
		require.NoError(
			t,
			os.WriteFile(tmp, []byte("new content"), 0o644), //nolint:gosec // test helper writing temp file
		)

		body := "before\n<user>\nold content\n</user>\nafter"
		got, err := markdown.MergeUserAgents(body, tmp)
		require.NoError(t, err)
		assert.Equal(t, "before\n<user>\nnew content\n</user>\nafter", got)
	})

	t.Run("appends when no marker found", func(t *testing.T) {
		t.Parallel()
		tmp := filepath.Join(t.TempDir(), "USER_AGENTS.md")
		require.NoError(
			t,
			os.WriteFile(tmp, []byte("user stuff"), 0o644), //nolint:gosec // test helper writing temp file
		)

		got, err := markdown.MergeUserAgents("body text", tmp)
		require.NoError(t, err)
		assert.Equal(t, "body text\n\nuser stuff\n", got)
	})
}
