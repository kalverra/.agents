package ticket

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTicket_ToFetchPayload_nilCommentsBecomesEmptySlice(t *testing.T) {
	t.Parallel()
	tk := &Ticket{ID: "1", Title: "T"}
	payload := tk.ToFetchPayload()
	assert.NotNil(t, payload.Comments)
	assert.Empty(t, payload.Comments)

	b, err := json.Marshal(payload)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"comments":[]`)
}
