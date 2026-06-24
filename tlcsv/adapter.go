package tlcsv

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/interline-io/transitland-lib/request"
	"github.com/twpayne/go-geom/encoding/geojson"
)

// Adapter provides an interface for working with various kinds of GTFS sources: zip, directory, url.
type Adapter interface {
	OpenFile(string, func(io.Reader)) error
	ReadRows(string, func(Row)) error
	Open() error
	Close() error
	Exists() bool
	Path() string
	SHA1() (string, error)
	DirSHA1() (string, error)
	String() string
}

// WriterAdapter provides a writing interface.
type WriterAdapter interface {
	WriteRows(string, [][]string) error
	WriteFeatures(string, []*geojson.Feature) error
	Adapter
}

// NewStoreAdapter is a convenience method for getting a GTFS Zip reader from the store.
func NewStoreAdapter(ctx context.Context, storage string, key string, fragment string) (*ZipAdapter, error) {
	store, err := request.GetStore(storage)
	if err != nil {
		return nil, err
	}
	r, _, err := store.Download(ctx, key)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return NewTmpZipAdapterFromReader(r, fragment)
}

// NewAdapter returns a basic adapter for the given URL.
// Use NewURLAdapter() to provide additional options.
func NewAdapter(address string) (Adapter, error) {
	parsedUrl, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	var a Adapter
	switch parsedUrl.Scheme {
	case "http":
		a = &URLAdapter{url: address}
	case "https":
		a = &URLAdapter{url: address}
	case "ftp":
		a = &URLAdapter{url: address}
	case "s3":
		a = &URLAdapter{url: address}
	case "overlay":
		a = NewOverlayAdapter(address)
	default:
		if fi, err := os.Stat(address); err == nil && fi.IsDir() {
			a = NewDirAdapter(address)
		} else {
			a = NewZipAdapter(address)
		}
	}
	return a, nil
}

// isFeedFile reports whether a top-level archive entry is a feed file: not a
// directory, not a dotfile, and not nested in a subdirectory (relative to the feed
// root). Shared by the directory and zip FileInfos implementations.
func isFeedFile(name string, isDir bool) bool {
	return !isDir && !strings.HasPrefix(name, ".") && !strings.Contains(name, "/")
}

// dirSHA1 concatenates the sorted, lowercase .txt feed files — each read through
// openFile — into a single SHA1. It is the shared DirSHA1 of the directory, zip, and
// overlay adapters; names are the feed-file base names (e.g. from FileInfos).
func dirSHA1(names []string, openFile func(string, func(io.Reader)) error) (string, error) {
	sort.Strings(names)
	h := sha1.New()
	for _, name := range names {
		if strings.HasPrefix(name, ".") || name != strings.ToLower(name) || !strings.HasSuffix(name, ".txt") {
			continue
		}
		var copyErr error
		if err := openFile(name, func(r io.Reader) { _, copyErr = io.Copy(h, r) }); err != nil {
			return "", err
		}
		if copyErr != nil {
			return "", copyErr
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// findInternalPrefix returns the directory containing stops.txt (the feed root)
// among the given entry paths — slash-separated, directories ending in "/"; "" for a
// flat archive or directory, and an error when more than one candidate is found.
// Adapter-agnostic: the zip adapters pass their entry names, a directory adapter its
// walked paths.
func findInternalPrefix(paths []string) (string, error) {
	prefixes := []string{}
	for _, fn := range paths {
		if strings.HasSuffix(fn, "/") || strings.HasPrefix(fn, ".") {
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

// WalkFile is one entry yielded by Walk: its Path relative to the walk prefix
// ("/"-separated, directories ending in "/") plus the entry's FileInfo.
type WalkFile struct {
	Path string
	fs.FileInfo
}

// walkable adapters enumerate their entries read-only, independent of Open. It is
// the single listing primitive FileInfos and feed-root detection are built on.
type walkable interface {
	Walk(prefix string) iter.Seq2[WalkFile, error]
}

// feedFileInfos collects the top-level feed files under prefix — excluding
// directories, dotfiles and nested entries — sorted by name.
func feedFileInfos(w walkable, prefix string) ([]os.FileInfo, error) {
	ret := []os.FileInfo{}
	for wf, err := range w.Walk(prefix) {
		if err != nil {
			return nil, err
		}
		if isFeedFile(wf.Path, wf.IsDir()) {
			ret = append(ret, wf.FileInfo)
		}
	}
	sort.Slice(ret, func(i, j int) bool { return ret[i].Name() < ret[j].Name() })
	return ret, nil
}

// detectInternalPrefix walks every entry and returns the feed root (the directory
// containing stops.txt); "" for a flat source, an error on ambiguity.
func detectInternalPrefix(w walkable) (string, error) {
	var paths []string
	for wf, err := range w.Walk("") {
		if err != nil {
			return "", err
		}
		paths = append(paths, wf.Path)
	}
	return findInternalPrefix(paths)
}
