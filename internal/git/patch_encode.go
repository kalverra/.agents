package git

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	fdiff "github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/utils/diff"
	dmp "github.com/sergi/go-diff/diffmatchpatch"
)

type contentPatch struct {
	filePatches []fdiff.FilePatch
}

func (p contentPatch) FilePatches() []fdiff.FilePatch { return p.filePatches }
func (p contentPatch) Message() string                { return "" }

type patchFile struct {
	path    string
	mode    filemode.FileMode
	content string
}

func (f *patchFile) Hash() plumbing.Hash {
	return plumbing.ComputeHash(plumbing.BlobObject, []byte(f.content))
}

func (f *patchFile) Mode() filemode.FileMode { return f.mode }
func (f *patchFile) Path() string            { return f.path }

type patchFilePatch struct {
	from, to fdiff.File
	chunks   []fdiff.Chunk
}

// IsBinary satisfies fdiff.FilePatch; binary files are handled before patch encoding.
func (p patchFilePatch) IsBinary() bool { return false }

func (p patchFilePatch) Files() (fdiff.File, fdiff.File) {
	return p.from, p.to
}

func (p patchFilePatch) Chunks() []fdiff.Chunk { return p.chunks }

type patchChunk struct {
	content string
	op      fdiff.Operation
}

func (c patchChunk) Content() string       { return c.content }
func (c patchChunk) Type() fdiff.Operation { return c.op }

func unifiedPatch(path, oldContent, newContent string, contextLines int) (string, int, int) {
	filePatch := buildFilePatch(path, oldContent, newContent)
	additions, deletions := filePatchStats(filePatch)

	var buf bytes.Buffer
	enc := fdiff.NewUnifiedEncoder(&buf, contextLines)
	if err := enc.Encode(contentPatch{filePatches: []fdiff.FilePatch{filePatch}}); err != nil {
		return fmt.Sprintf("malformed patch: %s", err), additions, deletions
	}

	return buf.String(), additions, deletions
}

func buildFilePatch(path, oldContent, newContent string) fdiff.FilePatch {
	mode := filemode.Regular
	var from, to fdiff.File

	if oldContent != "" {
		from = &patchFile{path: path, mode: mode, content: oldContent}
	}
	if newContent != "" {
		to = &patchFile{path: path, mode: mode, content: newContent}
	}

	chunks := textDiffChunks(oldContent, newContent)
	return patchFilePatch{from: from, to: to, chunks: chunks}
}

func textDiffChunks(oldContent, newContent string) []fdiff.Chunk {
	chunks := make([]fdiff.Chunk, 0, 8)
	for _, d := range diff.Do(oldContent, newContent) {
		var op fdiff.Operation
		switch d.Type {
		case dmp.DiffEqual:
			op = fdiff.Equal
		case dmp.DiffDelete:
			op = fdiff.Delete
		case dmp.DiffInsert:
			op = fdiff.Add
		}
		chunks = append(chunks, patchChunk{content: d.Text, op: op})
	}
	return chunks
}

func filePatchStats(fp fdiff.FilePatch) (additions, deletions int) {
	for _, chunk := range fp.Chunks() {
		s := chunk.Content()
		if len(s) == 0 {
			continue
		}
		lines := strings.Count(s, "\n")
		if s[len(s)-1] != '\n' {
			lines++
		}
		switch chunk.Type() {
		case fdiff.Add:
			additions += lines
		case fdiff.Delete:
			deletions += lines
		}
	}
	return additions, deletions
}
