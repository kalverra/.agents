package ticket

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestADFToMarkdown_nil(t *testing.T) {
	t.Parallel()
	assert.Empty(t, adfToMarkdown(nil))
}

func TestADFToMarkdown_paragraph(t *testing.T) {
	t.Parallel()
	raw := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{"type": "text", "text": "Hello"},
					{"type": "text", "text": " world"}
				]
			}
		]
	}`
	var v any
	require.NoError(t, json.Unmarshal([]byte(raw), &v))
	assert.Equal(t, "Hello world", adfToMarkdown(v))
}

func TestADFToMarkdown_headingLevels(t *testing.T) {
	t.Parallel()
	raw := `{
		"type": "doc",
		"version": 1,
		"content": [
			{"type": "heading", "attrs": {"level": 1}, "content": [{"type": "text", "text": "One"}]},
			{"type": "heading", "attrs": {"level": 3}, "content": [{"type": "text", "text": "Three"}]}
		]
	}`
	var v any
	require.NoError(t, json.Unmarshal([]byte(raw), &v))
	assert.Equal(t, "# One\n\n### Three", adfToMarkdown(v))
}

func TestADFToMarkdown_textMarks(t *testing.T) {
	t.Parallel()
	raw := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{"type": "text", "text": "bold", "marks": [{"type": "strong"}]},
					{"type": "text", "text": " "},
					{"type": "text", "text": "italic", "marks": [{"type": "em"}]},
					{"type": "text", "text": " "},
					{"type": "text", "text": "code", "marks": [{"type": "code"}]},
					{"type": "text", "text": " "},
					{"type": "text", "text": "link", "marks": [{"type": "link", "attrs": {"href": "https://a.test/x"}}]}
				]
			}
		]
	}`
	var v any
	require.NoError(t, json.Unmarshal([]byte(raw), &v))
	assert.Equal(t, "**bold** *italic* `code` [link](https://a.test/x)", adfToMarkdown(v))
}

func TestADFToMarkdown_bulletList(t *testing.T) {
	t.Parallel()
	raw := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "bulletList",
				"content": [
					{
						"type": "listItem",
						"content": [
							{
								"type": "paragraph",
								"content": [{"type": "text", "text": "First"}]
							}
						]
					},
					{
						"type": "listItem",
						"content": [
							{
								"type": "paragraph",
								"content": [{"type": "text", "text": "Second"}]
							}
						]
					}
				]
			}
		]
	}`
	var v any
	require.NoError(t, json.Unmarshal([]byte(raw), &v))
	assert.Equal(t, "- First\n- Second", adfToMarkdown(v))
}

func TestADFToMarkdown_orderedList(t *testing.T) {
	t.Parallel()
	raw := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "orderedList",
				"attrs": {"order": 1},
				"content": [
					{
						"type": "listItem",
						"content": [
							{"type": "paragraph", "content": [{"type": "text", "text": "A"}]}
						]
					},
					{
						"type": "listItem",
						"content": [
							{"type": "paragraph", "content": [{"type": "text", "text": "B"}]}
						]
					}
				]
			}
		]
	}`
	var v any
	require.NoError(t, json.Unmarshal([]byte(raw), &v))
	assert.Equal(t, "1. A\n2. B", adfToMarkdown(v))
}

func TestADFToMarkdown_codeBlock(t *testing.T) {
	t.Parallel()
	raw := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "codeBlock",
				"attrs": {"language": "go"},
				"content": [{"type": "text", "text": "package main\n"}]
			}
		]
	}`
	var v any
	require.NoError(t, json.Unmarshal([]byte(raw), &v))
	assert.Equal(t, "```go\npackage main\n```", adfToMarkdown(v))
}

func TestADFToMarkdown_rule(t *testing.T) {
	t.Parallel()
	raw := `{
		"type": "doc",
		"version": 1,
		"content": [
			{"type": "paragraph", "content": [{"type": "text", "text": "above"}]},
			{"type": "rule"},
			{"type": "paragraph", "content": [{"type": "text", "text": "below"}]}
		]
	}`
	var v any
	require.NoError(t, json.Unmarshal([]byte(raw), &v))
	assert.Equal(t, "above\n\n---\n\nbelow", adfToMarkdown(v))
}

func TestADFToMarkdown_blockquote(t *testing.T) {
	t.Parallel()
	raw := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "blockquote",
				"content": [
					{
						"type": "paragraph",
						"content": [{"type": "text", "text": "Quoted"}]
					}
				]
			}
		]
	}`
	var v any
	require.NoError(t, json.Unmarshal([]byte(raw), &v))
	assert.Equal(t, "> Quoted", adfToMarkdown(v))
}

func TestADFToMarkdown_hardBreak(t *testing.T) {
	t.Parallel()
	raw := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{"type": "text", "text": "line one"},
					{"type": "hardBreak"},
					{"type": "text", "text": "line two"}
				]
			}
		]
	}`
	var v any
	require.NoError(t, json.Unmarshal([]byte(raw), &v))
	assert.Equal(t, "line one  \nline two", adfToMarkdown(v))
}

func TestADFToMarkdown_paragraphTextAndInlineCard(t *testing.T) {
	t.Parallel()
	raw := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{"type": "text", "text": "figure out why here: "},
					{"type": "inlineCard", "attrs": {"url": "https://example.slack.com/archives/C09/p123"}}
				]
			}
		]
	}`
	var v any
	require.NoError(t, json.Unmarshal([]byte(raw), &v))
	assert.Equal(t, "figure out why here: <https://example.slack.com/archives/C09/p123>", adfToMarkdown(v))
}

func TestADFToMarkdown_docBlockInlineCard(t *testing.T) {
	t.Parallel()
	raw := `{
		"type": "doc",
		"version": 1,
		"content": [
			{"type": "paragraph", "content": [{"type": "text", "text": "before"}]},
			{"type": "inlineCard", "attrs": {"url": "https://thread.example/t/1"}}
		]
	}`
	var v any
	require.NoError(t, json.Unmarshal([]byte(raw), &v))
	assert.Equal(t, "before\n\n<https://thread.example/t/1>", adfToMarkdown(v))
}

func TestADFToMarkdown_textEmptyLinkUsesHref(t *testing.T) {
	t.Parallel()
	raw := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{"type": "text", "text": "see "},
					{
						"type": "text",
						"text": "",
						"marks": [{"type": "link", "attrs": {"href": "https://docs.example/x"}}]
					}
				]
			}
		]
	}`
	var v any
	require.NoError(t, json.Unmarshal([]byte(raw), &v))
	assert.Equal(t, "see <https://docs.example/x>", adfToMarkdown(v))
}

func TestADFToMarkdown_inlineCardWithTitle(t *testing.T) {
	t.Parallel()
	raw := `{
		"type": "doc",
		"version": 1,
		"content": [
			{
				"type": "paragraph",
				"content": [
					{
						"type": "inlineCard",
						"attrs": {
							"url": "https://smartcontract-it.atlassian.net/browse/DX-1",
							"title": "DX-1: Fix it"
						}
					}
				]
			}
		]
	}`
	var v any
	require.NoError(t, json.Unmarshal([]byte(raw), &v))
	assert.Equal(t, "[DX-1: Fix it](https://smartcontract-it.atlassian.net/browse/DX-1)", adfToMarkdown(v))
}
