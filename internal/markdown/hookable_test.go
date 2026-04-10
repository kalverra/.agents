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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := markdown.StripHookableDelimiterLines(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
