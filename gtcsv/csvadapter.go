package gtcsv

import (
	"archive/zip"
	"encoding/csv"
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
func NewAdapter(path string) ReaderAdapter {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return &URLAdapter{URL: path}
	} else if strings.HasSuffix(path, ".zip") {
		return &ZipAdapter{path}
	}
	return &DirAdapter{path}
}

func NewWriterAdapter(path string) WriterAdapter {
	return nil
}

// Adapter provides an interface for working with various kinds of GTFS sources: zip, directory, url.
type Adapter interface {
	Close() error
	Exists() bool
	Path() string
}

type ReaderAdapter interface {
	Open() error
	OpenFile(string, func(io.Reader)) error
	ReadRows(string, func(Row)) error
	Adapter
}

type WriterAdapter interface {
	Create() error
	WriteRow(string, []string) error
	Adapter
}

/////////////////////

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

/////////////////////

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

/////////////////////

type ZipWriterAdapter struct {
	path  string
	files map[string]*os.File
	out   *zip.Writer
}

func NewZipWriterAdapter(path string) *ZipWriterAdapter {
	return &ZipWriterAdapter{
		path:  path,
		files: map[string]*os.File{},
	}
}

func (adapter ZipWriterAdapter) Create() error {
	if adapter.Exists() {
		return errors.New("file already exists")
	}
	outf, err := os.Create(adapter.path)
	if err != nil {
		return err
	}
	adapter.out = zip.NewWriter(outf)
	return nil
}

func (adapter ZipWriterAdapter) Close() error {
	// Add files to zip
	for k, f := range adapter.files {
		zf, err := adapter.out.Create(k)
		if err != nil {
			return err
		}
		f.Seek(0, 0)
		if _, err := io.Copy(zf, f); err != nil {
			return err
		}
		f.Close()
		os.Remove(f.Name())
	}
	if err := adapter.out.Close(); err != nil {
		return err
	}
	return nil
}

func (adapter ZipWriterAdapter) Exists() bool {
	// Is the path a directory
	_, err := os.Stat(adapter.path)
	if err != nil {
		return false
	}
	return true
}

func (adapter ZipWriterAdapter) Path() string {
	return adapter.path
}

func (adapter ZipWriterAdapter) WriteRow(filename string, row []string) error {
	// Check if we have open fd
	outf, ok := adapter.files[filename]
	if !ok {
		var err2 error
		outf, err2 = ioutil.TempFile("", filename)
		if err2 != nil {
			return err2
		}
		adapter.files[filename] = outf
	}
	//
	csvw := csv.NewWriter(outf)
	csvw.Write(row)
	if err := csvw.Error(); err != nil {
		return err
	}
	return nil
}

/////////////////////

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
