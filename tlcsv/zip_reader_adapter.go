package tlcsv

import (
	"archive/zip"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"strings"

	"github.com/interline-io/transitland-lib/causes"
)

// ZipReaderAdapter reads a GTFS zip from memory (any io.ReaderAt) instead of a
// path on disk, decompressing each file on demand. Use it when the archive bytes
// are already in memory (e.g. an upload) and there is no usable filesystem —
// notably under js/wasm, where the path-based ZipAdapter cannot create temp files.
type ZipReaderAdapter struct {
	r              io.ReaderAt
	size           int64
	zr             *zip.Reader
	internalPrefix string
}

var _ Adapter = (*ZipReaderAdapter)(nil)

// NewZipReaderAdapter builds an adapter over a zip read from r (size bytes). The
// source must stay valid for the adapter's lifetime (it is read lazily).
func NewZipReaderAdapter(r io.ReaderAt, size int64) (*ZipReaderAdapter, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, err
	}
	return &ZipReaderAdapter{r: r, size: size, zr: zr}, nil
}

// NewZipReaderAdapterWithPrefix is like NewZipReaderAdapter but pins an explicit
// internal feed-root prefix (the subdirectory holding the feed) instead of
// auto-discovering one in Open — for an archive that wraps the feed in a known
// subdirectory, or whose root would otherwise be ambiguous. An empty prefix
// auto-discovers, exactly like NewZipReaderAdapter.
func NewZipReaderAdapterWithPrefix(r io.ReaderAt, size int64, internalPrefix string) (*ZipReaderAdapter, error) {
	adapter, err := NewZipReaderAdapter(r, size)
	if err != nil {
		return nil, err
	}
	adapter.internalPrefix = internalPrefix
	return adapter, nil
}

func (adapter *ZipReaderAdapter) String() string { return "memory.zip" }
func (adapter *ZipReaderAdapter) Path() string   { return "memory.zip" }
func (adapter *ZipReaderAdapter) Close() error   { return nil }

// Exists reports whether the archive parsed and holds any entries.
func (adapter *ZipReaderAdapter) Exists() bool {
	return adapter.zr != nil && len(adapter.zr.File) > 0
}

// Open auto-discovers the internal feed-root prefix (the directory containing
// stops.txt), mirroring ZipAdapter.
func (adapter *ZipReaderAdapter) Open() error {
	if !adapter.Exists() {
		return errors.New("file does not exist or invalid data")
	}
	if adapter.internalPrefix == "" {
		pfx, err := detectInternalPrefix(adapter)
		if err != nil {
			return err
		}
		adapter.internalPrefix = pfx
	}
	return nil
}

// Walk yields each archive entry under prefix as (path relative to prefix, info).
// The zip namespace is flat, so Walk("") enumerates the whole archive — that is how
// feed-root detection sees nested entries.
func (adapter *ZipReaderAdapter) Walk(prefix string) iter.Seq2[WalkFile, error] {
	return func(yield func(WalkFile, error) bool) {
		for _, zf := range adapter.zr.File {
			name := zf.Name
			if prefix != "" {
				if !strings.HasPrefix(name, prefix+"/") {
					continue
				}
				name = strings.TrimPrefix(name, prefix+"/")
			}
			if name == "" {
				continue
			}
			if !yield(WalkFile{Path: name, FileInfo: zf.FileInfo()}, nil) {
				return
			}
		}
	}
}

// OpenFile streams the named entry's decompressed bytes to cb, decompressing on
// demand. Each call opens a fresh reader, so a file may be read multiple times.
func (adapter *ZipReaderAdapter) OpenFile(filename string, cb func(io.Reader)) error {
	return openFileInZip(adapter.zr.File, adapter.internalPrefix, filename, cb)
}

func (adapter *ZipReaderAdapter) ReadRows(filename string, cb func(Row)) error {
	return adapter.OpenFile(filename, func(in io.Reader) { ReadRows(in, cb) })
}

// SHA1 hashes the whole archive, streamed from the source.
func (adapter *ZipReaderAdapter) SHA1() (string, error) {
	h := sha1.New()
	if _, err := io.Copy(h, io.NewSectionReader(adapter.r, 0, adapter.size)); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (adapter *ZipReaderAdapter) DirSHA1() (string, error) {
	fis, err := adapter.FileInfos()
	if err != nil {
		return "", err
	}
	names := make([]string, len(fis))
	for i, fi := range fis {
		names[i] = fi.Name()
	}
	return dirSHA1(names, adapter.OpenFile)
}

// FileInfos returns an os.FileInfo for each top-level feed file under the internal
// feed-root prefix. Call after Open so the prefix is resolved.
func (adapter *ZipReaderAdapter) FileInfos() ([]os.FileInfo, error) {
	return feedFileInfos(adapter, adapter.internalPrefix)
}

// --- shared zip helpers (used by both ZipAdapter and ZipReaderAdapter) ---

// openFileInZip finds filename (under internalPrefix) among the zip entries and
// streams its decompressed bytes to cb; a missing file is an error and cb is not
// called.
func openFileInZip(files []*zip.File, internalPrefix, filename string, cb func(io.Reader)) error {
	want := filepath.Join(internalPrefix, filename)
	for _, f := range files {
		if f.Name != want {
			continue
		}
		in, err := f.Open()
		if err != nil {
			return err
		}
		defer in.Close()
		cb(in)
		return nil
	}
	return causes.NewFileNotPresentError(filename)
}
