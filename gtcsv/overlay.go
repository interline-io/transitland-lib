package gtcsv

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

// OverlayAdapter searches a specified list of directories for the specified file.
// Used for reducing the complexity of writing tests.
type OverlayAdapter struct {
	paths []string
}

// NewOverlayAdapter returns a new OverlayAdapter.
func NewOverlayAdapter(paths ...string) OverlayAdapter {
	return OverlayAdapter{paths: paths}
}

// SHA1 .
func (adapter OverlayAdapter) SHA1() (string, error) {
	return "", errors.New("cannot take SHA1 of directory")
}

// DirSHA1 .
func (adapter OverlayAdapter) DirSHA1() (string, error) {
	return "", errors.New("not supported")
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
	return adapter.OpenFile(filename, func(in io.Reader) {
		ReadRows(in, cb)
	})
}

// Open implements CSV Adapter Open.
func (adapter OverlayAdapter) Open() error {
	for _, path := range adapter.paths {
		if fi, err := os.Stat(path); err != nil || !fi.Mode().IsDir() {
			return errors.New("overlay path is not a directory")
		}
	}
	return nil
}

// Close implements CSV Adapter.Close.
func (adapter OverlayAdapter) Close() error {
	return nil
}

// Path implements CSV Adapter.Path.
func (adapter OverlayAdapter) Path() string {
	return adapter.paths[0]
}
