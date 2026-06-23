package tlcsv

import (
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
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

// NewZipReaderAdapterFromBytes builds an adapter over raw zip bytes. The slice is
// retained (read lazily), so the caller must not mutate it.
func NewZipReaderAdapterFromBytes(b []byte) (*ZipReaderAdapter, error) {
	return NewZipReaderAdapter(bytes.NewReader(b), int64(len(b)))
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
		pfx, err := findInternalPrefixInZip(adapter.zr.File)
		if err != nil {
			return err
		}
		adapter.internalPrefix = pfx
	}
	return nil
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
	return dirSHA1InZip(adapter.zr.File, adapter.internalPrefix)
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

// findInternalPrefixInZip returns the directory containing stops.txt (the feed
// root); "" for a flat archive, an error if more than one candidate is found.
func findInternalPrefixInZip(files []*zip.File) (string, error) {
	prefixes := []string{}
	for _, zf := range files {
		fi := zf.FileInfo()
		fn := zf.Name
		if fi.IsDir() || strings.HasPrefix(fn, ".") {
			continue
		}
		if filepath.Base(fn) == "stops.txt" {
			prefixes = append(prefixes, filepath.Dir(fn))
		}
	}
	if len(prefixes) > 1 {
		return "", errors.New("more than one valid prefix found")
	} else if len(prefixes) == 1 {
		pfx := prefixes[0]
		if pfx == "." {
			pfx = ""
		}
		return pfx, nil
	}
	return "", nil
}

// dirSHA1InZip hashes the sorted, concatenated top-level lowercase .txt entries
// (under internalPrefix), matching ZipAdapter/DirAdapter.
func dirSHA1InZip(files []*zip.File, internalPrefix string) (string, error) {
	sorted := append([]*zip.File(nil), files...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	h := sha1.New()
	for _, zf := range sorted {
		fi := zf.FileInfo()
		fn := zf.Name
		if internalPrefix != "" {
			fn = strings.Replace(zf.Name, internalPrefix+"/", "", 1)
		}
		if fi.IsDir() || strings.HasPrefix(fn, ".") || strings.Contains(fn, "/") {
			continue
		}
		if fi.Name() != strings.ToLower(fi.Name()) || !strings.HasSuffix(fi.Name(), ".txt") {
			continue
		}
		f, err := zf.Open()
		if err != nil {
			return "", err
		}
		_, err = io.Copy(h, f)
		f.Close()
		if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
