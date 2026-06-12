package git

import (
	"io"
	"os"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/index"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// fileSnapshot captures readable file state for diffing.
type fileSnapshot struct {
	exists bool
	binary bool
	hash   plumbing.Hash
	text   string
}

func (s fileSnapshot) equal(other fileSnapshot) bool {
	if s.exists != other.exists {
		return false
	}
	if !s.exists {
		return true
	}
	if s.binary || other.binary {
		return s.hash == other.hash
	}
	return s.text == other.text
}

func treeFileSnapshot(tree *object.Tree, path string) (fileSnapshot, error) {
	file, err := tree.File(path)
	if err == object.ErrFileNotFound {
		return fileSnapshot{}, nil
	}
	if err != nil {
		return fileSnapshot{}, err
	}

	isBinary, err := file.IsBinary()
	if err != nil {
		return fileSnapshot{}, err
	}
	if isBinary {
		return fileSnapshot{exists: true, binary: true, hash: file.Hash}, nil
	}

	content, err := file.Contents()
	if err != nil {
		return fileSnapshot{}, err
	}
	return fileSnapshot{exists: true, text: content}, nil
}

// effectiveFileSnapshot resolves the current file state from worktree, index, and HEAD.
// Staged-only changes use the index; worktree edits use disk content.
func effectiveFileSnapshot(
	repo *gogit.Repository,
	repoPath string,
	headTree *object.Tree,
	idxMap map[string]*index.Entry,
	status gogit.Status,
	path string,
) (fileSnapshot, error) {
	diskPath := joinRepoPath(repoPath, path)

	if st, ok := status[path]; ok {
		if st.Worktree == gogit.Deleted {
			return fileSnapshot{}, nil
		}
		if st.Staging != gogit.Unmodified && st.Worktree == gogit.Unmodified {
			if st.Staging == gogit.Deleted {
				if _, err := os.Stat(diskPath); err == nil {
					return diskSnapshot(diskPath)
				}
				return fileSnapshot{}, nil
			}
			if entry := idxMap[path]; entry != nil {
				return blobSnapshot(repo, entry.Hash)
			}
			return fileSnapshot{}, nil
		}
	}

	if _, err := os.Stat(diskPath); err == nil {
		return diskSnapshot(diskPath)
	}

	if entry := idxMap[path]; entry != nil {
		return blobSnapshot(repo, entry.Hash)
	}

	return treeFileSnapshot(headTree, path)
}

func diskSnapshot(path string) (fileSnapshot, error) {
	f, err := os.Open(path) //nolint:gosec // path is inside repo worktree
	if err != nil {
		return fileSnapshot{}, err
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, 8000)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return fileSnapshot{}, err
	}
	buf = buf[:n]

	if isBinaryBytes(buf) {
		stat, err := f.Stat()
		if err != nil {
			return fileSnapshot{}, err
		}

		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return fileSnapshot{}, err
		}

		h := plumbing.NewHasher(plumbing.BlobObject, stat.Size())
		if _, err := io.Copy(h, f); err != nil {
			return fileSnapshot{}, err
		}

		return fileSnapshot{
			exists: true,
			binary: true,
			hash:   h.Sum(),
		}, nil
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return fileSnapshot{}, err
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return fileSnapshot{}, err
	}
	return fileSnapshot{exists: true, text: string(data)}, nil
}

func blobSnapshot(repo *gogit.Repository, hash plumbing.Hash) (fileSnapshot, error) {
	blob, err := repo.BlobObject(hash)
	if err != nil {
		return fileSnapshot{}, err
	}

	reader, err := blob.Reader()
	if err != nil {
		return fileSnapshot{}, err
	}
	defer func() { _ = reader.Close() }()

	buf := make([]byte, 8000)
	n, err := io.ReadFull(reader, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return fileSnapshot{}, err
	}
	buf = buf[:n]

	if isBinaryBytes(buf) {
		return fileSnapshot{exists: true, binary: true, hash: hash}, nil
	}

	rest, err := io.ReadAll(reader)
	if err != nil {
		return fileSnapshot{}, err
	}

	text := make([]byte, 0, len(buf)+len(rest))
	text = append(text, buf...)
	text = append(text, rest...)

	return fileSnapshot{exists: true, text: string(text)}, nil
}

func indexMap(idx *index.Index) map[string]*index.Entry {
	m := make(map[string]*index.Entry, len(idx.Entries))
	for _, entry := range idx.Entries {
		m[entry.Name] = entry
	}
	return m
}
