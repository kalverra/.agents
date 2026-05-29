package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAntigravityPaths(t *testing.T) {
	// 1. Test environment variable override
	t.Setenv("ANTIGRAVITY_CONFIG_DIR", "/custom/antigravity/dir")
	require.Equal(t, "/custom/antigravity/dir", AntigravityConfigDir())

	// 2. Test default config dir (clearing the env var first)
	t.Setenv("ANTIGRAVITY_CONFIG_DIR", "")
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	expectedDefaultDir := filepath.Join(home, ".gemini")
	require.Equal(t, expectedDefaultDir, AntigravityConfigDir())

	// 3. Test MarkdownDest(Antigravity) uses GEMINI.md in the config dir
	expectedMarkdownDest := filepath.Join(expectedDefaultDir, "GEMINI.md")
	require.Equal(t, expectedMarkdownDest, MarkdownDest(Antigravity))

	// 4. Test SkillsDest(Antigravity) uses the config dir
	expectedSkillsDest := filepath.Join(expectedDefaultDir, "skills")
	require.Equal(t, expectedSkillsDest, SkillsDest(Antigravity))
}
