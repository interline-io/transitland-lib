package tlcsv

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// DirSHA1 returns the SHA1 of all the .txt files in the main directory, sorted, and concatenated.
func (adapter OverlayAdapter) DirSHA1() (string, error) {
	alltxts := map[string]string{}
	for _, path := range adapter.paths {
		f, err := os.Open(path)
		if err != nil {
			return "", err
		}
		fis, err := f.Readdir(-1)
		f.Close()
		if err != nil {
			return "", err
		}
		for _, fi := range fis {
			fn := fi.Name()
			if fi.IsDir() || !strings.HasSuffix(fn, ".txt") || strings.HasPrefix(fn, ".") || strings.Contains(fn, "/") {
				continue
			}
			if _, ok := alltxts[fn]; ok {
				continue
			}
			alltxts[fn] = filepath.Join(path, fn)
		}
	}
	keys := []string{}
	for k := range alltxts {
		keys = append(keys, k)
	}
	// Sort the files
	sort.Strings(keys)
	// Hash
	h := sha1.New()
	for _, k := range keys {
		f, err := os.Open(filepath.Join(alltxts[k]))
		if err != nil {
			return "", err
		}
		io.Copy(h, f)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
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
