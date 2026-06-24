package tlcsv

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/interline-io/log"
)

// OverlayAdapter searches a specified list of directories for the specified file.
// Used for reducing the complexity of writing tests.
type OverlayAdapter struct {
	paths []string
}

// NewOverlayAdapter returns a new OverlayAdapter.
func NewOverlayAdapter(paths ...string) OverlayAdapter {
	if len(paths) == 1 {
		firstPath := paths[0]
		paths = strings.Split(strings.TrimPrefix(firstPath, "overlay://"), ",")
	}
	return OverlayAdapter{paths: paths}
}

func (adapter OverlayAdapter) String() string {
	return "overlay"
}

func (adapter OverlayAdapter) Files() ([]string, error) {
	check := map[string]bool{}
	for _, path := range adapter.paths {
		files, err := filepath.Glob(filepath.Join(path, "*.txt"))
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			fn := filepath.Base(f)
			if check[fn] {
				continue
			}
			check[fn] = true
		}
	}
	files := []string{}
	for k := range check {
		files = append(files, k)
	}
	return files, nil
}

func (adapter OverlayAdapter) CreateZip(outfile string) error {
	outadapter := NewZipWriterAdapter(outfile)
	if err := adapter.Open(); err != nil {
		return err
	}
	files, err := adapter.Files()
	if err != nil {
		return err
	}
	for _, filename := range files {
		var err2 error
		err := adapter.OpenFile(filename, func(r io.Reader) {
			err2 = outadapter.AddFile(filename, r)
		})
		if err != nil {
			return err
		}
		if err2 != nil {
			return err
		}
	}
	return outadapter.Close()
}

// SHA1 is an alias for DirSHA1
func (adapter OverlayAdapter) SHA1() (string, error) {
	return adapter.DirSHA1()
}

// DirSHA1 returns the SHA1 of all the .txt files across the overlay paths (first
// path wins on conflict), sorted by name and concatenated.
func (adapter OverlayAdapter) DirSHA1() (string, error) {
	names, err := adapter.Files()
	if err != nil {
		return "", err
	}
	return dirSHA1(names, adapter.OpenFile)
}

// OpenFile searches paths until it finds the specified file.
func (adapter OverlayAdapter) OpenFile(filename string, cb func(io.Reader)) error {
	for _, fn := range adapter.paths {
		in, err := os.Open(filepath.Join(fn, filename))
		if err != nil {
			continue
		}
		defer in.Close()
		cb(in)
		return nil
	}
	return errors.New("file not found")
}

// ReadRows implements CSV Adapter ReadRows.
func (adapter OverlayAdapter) ReadRows(filename string, cb func(Row)) error {
	t0 := time.Now()
	log.For(context.TODO()).Trace().Str("filename", filename).Msg("tlcsv: read pass")
	err := adapter.OpenFile(filename, func(in io.Reader) {
		ReadRows(in, cb)
	})
	log.For(context.TODO()).Trace().Str("filename", filename).Int("elapsed_ms", int(time.Since(t0).Milliseconds())).Msg("tlcsv: read pass complete")
	return err
}

// Open implements CSV Adapter Open.
func (adapter OverlayAdapter) Open() error {
	return nil
}

// Close implements CSV Adapter.Close.
func (adapter OverlayAdapter) Close() error {
	return nil
}

// Path implements CSV Adapter.Path.
func (adapter OverlayAdapter) Path() string {
	return fmt.Sprintf("overlay://%s", strings.Join(adapter.paths, ","))
}

// Exists implements CSV Adapter.Exists.
func (adapter OverlayAdapter) Exists() bool {
	for _, path := range adapter.paths {
		if fi, err := os.Stat(path); err != nil || !fi.Mode().IsDir() {
			return false
		}
	}
	return true
}
