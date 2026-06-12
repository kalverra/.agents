package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitOutputIncludesStderr(t *testing.T) {
	t.Parallel()

	// Run git in a non-repo directory — git writes a useful message to stderr.
	dir := t.TempDir()
	_, err := gitOutput(dir, "rev-parse", "--show-toplevel")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repository", "error should include git's stderr, not just exit code")
}
