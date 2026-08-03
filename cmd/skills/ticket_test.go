package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kalverra/agents/internal/ticket"
)

func TestTicketStatusAutoWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	key := "DX-999"
	file := filepath.Join(tmpDir, key+".md")

	tk := &ticket.Ticket{
		ID:          key,
		Title:       "Auto write test ticket",
		Description: "Test description",
		Status:      "In Progress",
		URL:         "https://example/browse/DX-999",
	}

	content := ticket.TicketToMarkdownFile(tk)
	err := os.WriteFile(file, []byte(content), 0644)
	require.NoError(t, err)

	data, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Contains(
		t,
		string(data),
		`<ticket summary="Auto write test ticket" link="https://example/browse/DX-999" id="DX-999" status="In Progress">`,
	)
}

func TestFormatBranchDefaultIndication(t *testing.T) {
	t.Parallel()

	gotDefault := formatBranchOutput("main", "main")
	assert.Equal(t, "main (default)", gotDefault)

	gotNonDefault := formatBranchOutput("add-widget/DX-123", "main")
	assert.Equal(t, "add-widget/DX-123", gotNonDefault)
}
