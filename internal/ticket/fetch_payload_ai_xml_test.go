package ticket

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchPayloadToAIXML_structure(t *testing.T) {
	t.Parallel()
	p := FetchPayload{
		Task: TaskInfo{
			ID:          "DX-1",
			Title:       "Hello",
			Description: "Line1\n\nLine2",
			Status:      "Open",
			URL:         "https://x/browse/DX-1",
		},
		Comments: []TaskComment{
			{ID: "c1", Content: "Note", PostedAt: "2026-01-01", ProjectID: "p9"},
		},
	}
	got := FetchPayloadToAIXML(p)
	require.Contains(t, got, "<ticket_fetch")
	require.Contains(t, got, `status="ok"`)
	require.Contains(t, got, "<id>DX-1</id>")
	require.Contains(t, got, "<title>Hello</title>")
	require.Contains(t, got, "<description>")
	require.Contains(t, got, "Line1")
	require.Contains(t, got, "</description>")
	require.Contains(t, got, "<status>Open</status>")
	require.Contains(t, got, "<url>https://x/browse/DX-1</url>")
	require.Contains(t, got, "<comments>")
	require.Contains(t, got, "<comment>")
	require.Contains(t, got, "<id>c1</id>")
	require.Contains(t, got, "<posted_at>2026-01-01</posted_at>")
	require.Contains(t, got, "<project_id>p9</project_id>")
	require.Contains(t, got, "<content>Note</content>")
	require.Contains(t, got, "</ticket_fetch>")
}

func TestFetchPayloadToAIXML_escapesXML(t *testing.T) {
	t.Parallel()
	p := FetchPayload{
		Task: TaskInfo{
			ID:          "A&B",
			Title:       `1 < 2 & "3"`,
			Description: "]]>",
			Status:      "",
			URL:         "",
		},
		Comments: []TaskComment{
			{ID: "x", Content: "<tag>", PostedAt: ""},
		},
	}
	got := FetchPayloadToAIXML(p)
	assert.Contains(t, got, "<id>A&amp;B</id>")
	assert.Contains(t, got, "<title>1 &lt; 2 &amp; &quot;3&quot;</title>")
	assert.Contains(t, got, "<description>]]&gt;</description>")
	assert.Contains(t, got, "<content>&lt;tag&gt;</content>")
}

func TestFetchPayloadToAIXML_omitsEmptyOptionalFields(t *testing.T) {
	t.Parallel()
	p := FetchPayload{
		Task: TaskInfo{
			ID: "1", Title: "T", Description: "", Status: "", URL: "",
		},
		Comments: []TaskComment{{ID: "c", Content: "x", PostedAt: "", ProjectID: ""}},
	}
	got := FetchPayloadToAIXML(p)
	assert.NotContains(t, got, "<description>")
	assert.NotContains(t, got, "<status>")
	assert.NotContains(t, got, "<url>")
	assert.NotContains(t, got, "<posted_at>")
	assert.NotContains(t, got, "<project_id>")
}
