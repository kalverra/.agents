package markdown_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kalverra/agents/internal/markdown"
)

func TestStripHookableSections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no hookable sections",
			input: "hello\nworld\n",
			want:  "hello\nworld\n",
		},
		{
			name: "single section removed",
			input: `before
<hookable name="rtk">
some content
more content
</hookable>
after
`,
			want: `before
after
`,
		},
		{
			name: "multiple sections removed",
			input: `top
<hookable name="a">
first
</hookable>
middle
<hookable name="b">
second
</hookable>
bottom
`,
			want: `top
middle
bottom
`,
		},
		{
			name: "inline single-line block removed (matches GLOBAL_AGENTS.md format)",
			input: `<tools>
<hookable name="rtk"><rule>Prepend "rtk" to ALL shell commands.</rule></hookable>
<rule name="documentation">docs</rule>
</tools>
`,
			want: `<tools>
<rule name="documentation">docs</rule>
</tools>
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := markdown.StripHookableSections(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStripHookableDelimiterLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no hookable tags",
			input: "hello\nworld\n",
			want:  "hello\nworld\n",
		},
		{
			name: "tags removed but content kept",
			input: `before
<hookable name="rtk">
some content
</hookable>
after
`,
			want: `before
some content
after
`,
		},
		{
			name:  "inline tags removed but content kept",
			input: `<hookable name="rtk"><rule>Prepend "rtk" to ALL shell commands.</rule></hookable>`,
			want:  `<rule>Prepend "rtk" to ALL shell commands.</rule>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := markdown.StripHookableDelimiterLines(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
