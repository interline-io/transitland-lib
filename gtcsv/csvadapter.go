package gtcsv

import (
	"archive/zip"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/interline-io/gotransit/causes"
	"github.com/interline-io/gotransit/internal/log"
)

// NewAdapter returns an Adapter based on the provided path or url.
func NewAdapter(path string) Adapter {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return &URLAdapter{URL: path}
	} else if strings.HasSuffix(path, ".zip") {
		return &ZipAdapter{path}
	}
	return &DirAdapter{path}
}

// Adapter provides an interface for working with various kinds of GTFS sources: zip, directory, url.
type Adapter interface {
	OpenFile(string, func(io.Reader)) error
	ReadRows(string, func(Row)) error
	Open() error
	Close() error
	Exists() bool
	Path() string
}

// URLAdapter downloads a GTFS URL to a temporary file, and removes the file when it is closed.
type URLAdapter struct {
	URL string
	ZipAdapter
}

// Open the adapter, and download the provided URL to a temporary file.
func (adapter *URLAdapter) Open() error {
	// Get the data
	resp, err := http.Get(adapter.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Create the file
	tmpfile, err := ioutil.TempFile("", "gtfs.zip")
	if err != nil {
		return err
	}
	defer tmpfile.Close()
	// Get the full path
	tmpfilepath := tmpfile.Name()
	// Write the body to file
	_, err = io.Copy(tmpfile, resp.Body)
	log.Debug("Downloaded %s to %s", adapter.URL, tmpfilepath)
	adapter.ZipAdapter = ZipAdapter{path: tmpfilepath}
	return nil
}

// Close the adapter, and remove the temporary file. An error is returned if the file could not be deleted.
func (adapter *URLAdapter) Close() error {
	return os.Remove(adapter.ZipAdapter.path)
}

// ZipAdapter supports zip archives.
type ZipAdapter struct {
	path string
}

// Open the adapter. Return an error if the file does not exist.
func (adapter ZipAdapter) Open() error {
	if !adapter.Exists() {
		return errors.New("file does not exist")
	}
	return nil
}

// Close the adapter.
func (adapter ZipAdapter) Close() error {
	return nil
}

// Path returns the path to the zip file.
func (adapter ZipAdapter) Path() string {
	return adapter.path
}

// Exists returns if the zip file exists.
func (adapter ZipAdapter) Exists() bool {
	// Is the file readable
	r, err := zip.OpenReader(adapter.path)
	if err != nil {
		return false
	}
	r.Close()
	return true
}

// OpenFile opens the file inside the archive.
func (adapter ZipAdapter) OpenFile(filename string, cb func(io.Reader)) error {
	r, err := zip.OpenReader(adapter.path)
	if err != nil {
		return err
	}
	defer r.Close()
	var inFile *zip.File
	for _, f := range r.File {
		if f.Name != filename {
			continue
		}
		inFile = f
	}
	if inFile == nil {
		return causes.NewFileNotPresentError(filename)
	}
	//
	in, err := inFile.Open()
	defer in.Close()
	if err != nil {
		return err
	}
	cb(in)
	return nil
}

// ReadRows opens the specified file and runs the callback on each Row. An error is returned if the file cannot be read.
func (adapter ZipAdapter) ReadRows(filename string, cb func(Row)) error {
	return adapter.OpenFile(filename, func(in io.Reader) {
		ReadRows(in, cb)
	})
}

// DirAdapter supports plain directories of CSV files.
type DirAdapter struct {
	path string
}

// Open the adapter. Return an error if the directory does not exist.
func (adapter DirAdapter) Open() error {
	if !adapter.Exists() {
		return errors.New("file does not exist")
	}
	return nil
}

// Close the adapter.
func (adapter DirAdapter) Close() error {
	return nil
}

// Path returns the directory path.
func (adapter DirAdapter) Path() string {
	return adapter.path
}

// OpenFile opens a file in the directory. Returns an error if the file cannot be read.
func (adapter DirAdapter) OpenFile(filename string, cb func(io.Reader)) error {
	in, err := os.Open(filepath.Join(adapter.path, filename))
	if err != nil {
		return err
	}
	defer in.Close()
	cb(in)
	return nil
}

// ReadRows opens the file and runs the callback for each row. An error is returned if the file cannot be read.
func (adapter DirAdapter) ReadRows(filename string, cb func(Row)) error {
	return adapter.OpenFile(filename, func(in io.Reader) {
		ReadRows(in, cb)
	})
}

// Exists checks if the specified directory exists.
func (adapter DirAdapter) Exists() bool {
	// Is the path a directory
	fi, err := os.Stat(adapter.path)
	if err != nil {
		return false
	}
	return fi.Mode().IsDir()
}
