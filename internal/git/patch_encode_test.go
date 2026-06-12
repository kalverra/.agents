package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnifiedPatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		oldContent  string
		newContent  string
		wantAdd     int
		wantDel     int
		wantInPatch []string
	}{
		{
			name:        "add line",
			oldContent:  "one\n",
			newContent:  "one\ntwo\n",
			wantAdd:     1,
			wantDel:     0,
			wantInPatch: []string{"+two"},
		},
		{
			name:        "delete line",
			oldContent:  "one\ntwo\n",
			newContent:  "one\n",
			wantAdd:     0,
			wantDel:     1,
			wantInPatch: []string{"-two"},
		},
		{
			name:        "modify line",
			oldContent:  "old\n",
			newContent:  "new\n",
			wantAdd:     1,
			wantDel:     1,
			wantInPatch: []string{"-old", "+new"},
		},
		{
			name:        "new file",
			oldContent:  "",
			newContent:  "hello\n",
			wantAdd:     1,
			wantDel:     0,
			wantInPatch: []string{"+hello"},
		},
		{
			name:        "deleted file",
			oldContent:  "gone\n",
			newContent:  "",
			wantAdd:     0,
			wantDel:     1,
			wantInPatch: []string{"-gone"},
		},
		{
			name:       "no trailing newline",
			oldContent: "alpha",
			newContent: "beta",
			wantAdd:    1,
			wantDel:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			patch, add, del := unifiedPatch("file.txt", tt.oldContent, tt.newContent, 3)
			assert.Equal(t, tt.wantAdd, add)
			assert.Equal(t, tt.wantDel, del)
			assert.NotEmpty(t, patch)
			for _, want := range tt.wantInPatch {
				assert.Contains(t, patch, want)
			}
		})
	}
}

func TestFilePatchStats(t *testing.T) {
	t.Parallel()

	fp := buildFilePatch("file.txt", "one\n", "one\ntwo\nthree\n")
	add, del := filePatchStats(fp)
	require.Equal(t, 2, add)
	require.Equal(t, 0, del)
}

func TestTextDiffChunks(t *testing.T) {
	t.Parallel()

	chunks := textDiffChunks("abc", "abx")
	require.NotEmpty(t, chunks)

	hasChange := false
	for _, c := range chunks {
		if c.Type() != 0 {
			hasChange = true
			break
		}
	}
	assert.True(t, hasChange)
}
