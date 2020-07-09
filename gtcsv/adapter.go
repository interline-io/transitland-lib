package gtcsv

import (
	"archive/zip"
	"crypto/sha1"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/interline-io/gotransit/causes"
	"github.com/interline-io/gotransit/internal/log"
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
}

// WriterAdapter provides a writing interface.
type WriterAdapter interface {
	WriteRows(string, [][]string) error
	Adapter
}

/////////////////////

// URLAdapter downloads a GTFS URL to a temporary file, and removes the file when it is closed.
type URLAdapter struct {
	url string
	ZipAdapter
}

// Open the adapter, and download the provided URL to a temporary file.
func (adapter *URLAdapter) Open() error {
	if adapter.ZipAdapter.path != "" {
		return nil // already open
	}
	// Get the data
	split := strings.SplitN(adapter.url, "#", 2)
	fragment := ""
	if len(split) > 1 {
		adapter.url = split[0]
		fragment = split[1]
	}
	resp, err := http.Get(adapter.url)
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
	log.Debug("Downloaded %s to %s", adapter.url, tmpfilepath)
	checkzip := tmpfilepath
	if fragment != "" {
		checkzip = tmpfilepath + "#" + fragment
	}
	za := NewZipAdapter(checkzip)
	if za == nil {
		return errors.New("could not open")
	}
	adapter.ZipAdapter = *za
	adapter.ZipAdapter.tmpfilepath = tmpfilepath
	return adapter.ZipAdapter.Open()
}

// Close the adapter, and remove the temporary file. An error is returned if the file could not be deleted.
func (adapter *URLAdapter) Close() error {
	return adapter.ZipAdapter.Close()
}

/////////////////////

// S3Adapter downloads a GTFS file from an S3 bucket to a temporary file, and removes the file when it is closed.
type S3Adapter struct {
	url string
	ZipAdapter
}

// Open the adapter, and download the provided URL to a temporary file.
func (adapter *S3Adapter) Open() error {
	if adapter.ZipAdapter.path != "" {
		return nil // already open
	}
	// Create the file
	tmpfile, err := ioutil.TempFile("", "gtfs.zip")
	if err != nil {
		return err
	}
	defer tmpfile.Close()
	// Get the full path
	tmpfilepath := tmpfile.Name()
	// Download from S3 using the CLI
	awscmd := exec.Command("aws", "s3", "cp", adapter.url, tmpfilepath)
	if output, err := awscmd.Output(); err != nil {
		return fmt.Errorf("error downloading %s: %s %s", adapter.url, output, err.Error())
	}
	log.Debug("Downloaded %s to %s", adapter.url, tmpfilepath)
	adapter.ZipAdapter = ZipAdapter{path: tmpfilepath, tmpfilepath: tmpfilepath}
	return adapter.ZipAdapter.Open()
}

// Close the adapter, and remove the temporary file. An error is returned if the file could not be deleted.
func (adapter *S3Adapter) Close() error {
	return adapter.ZipAdapter.Close()
}

/////////////////////

// ZipAdapter supports zip archives.
type ZipAdapter struct {
	path           string
	internalPrefix string
	tmpfilepath    string
}

// NewZipAdapter returns an initialized zip adapter.
func NewZipAdapter(path string) *ZipAdapter {
	// Does this zip have a nested path?
	adapter := ZipAdapter{path: path}
	spliturl := strings.SplitN(adapter.path, "#", 2)
	if len(spliturl) > 1 {
		adapter.path = spliturl[0]
		adapter.internalPrefix = spliturl[1]
	}
	// Do we need to extract a nested zip.
	if strings.HasSuffix(adapter.internalPrefix, ".zip") {
		pf := adapter.internalPrefix
		adapter.internalPrefix = ""
		tmpfilepath := ""
		err := adapter.OpenFile(pf, func(r io.Reader) {
			// Create the file
			tmpfile, _ := ioutil.TempFile("", "gtfs.zip")
			defer tmpfile.Close()
			// Get the full path
			tmpfilepath = tmpfile.Name()
			// Write the body to file
			io.Copy(tmpfile, r)
			log.Debug("Extracted %s internal prefix %s to %s", adapter.path, adapter.internalPrefix, tmpfilepath)
		})
		if err != nil {
			return &adapter
		}
		adapter.path = tmpfilepath
		adapter.tmpfilepath = tmpfilepath
		adapter.internalPrefix = ""
	}
	return &adapter
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
	if adapter.tmpfilepath != "" {
		log.Debug("removing temp file: %s", adapter.tmpfilepath)
		if err := os.Remove(adapter.tmpfilepath); err != nil {
			return err
		}
	}
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
		if f.Name != filepath.Join(adapter.internalPrefix, filename) {
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

// SHA1 returns the SHA1 checksum of the zip archive.
func (adapter ZipAdapter) SHA1() (string, error) {
	f, err := os.Open(adapter.path)
	if err != nil {
		return "", err
	}
	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// DirSHA1 returns the SHA1 of all the .txt files in the main directory, sorted, and concatenated.
func (adapter ZipAdapter) DirSHA1() (string, error) {
	r, err := zip.OpenReader(adapter.path)
	if err != nil {
		return "", err
	}
	defer r.Close()
	// Sort the files
	sort.Slice(r.File, func(i, j int) bool { return r.File[i].Name < r.File[j].Name })
	// Generate SHA1
	h := sha1.New()
	for _, zf := range r.File {
		fi := zf.FileInfo()
		fn := zf.Name
		if adapter.internalPrefix != "" {
			fn = strings.Replace(zf.Name, adapter.internalPrefix+"/", "", 1) // remove internalPrefix
		}
		if fi.IsDir() || !strings.HasSuffix(fn, ".txt") || strings.HasPrefix(fn, ".") || strings.Contains(fn, "/") {
			continue
		}
		f, err := zf.Open()
		defer f.Close()
		if err != nil {
			return "", err
		}
		io.Copy(h, f)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

/////////////////////

// DirAdapter supports plain directories of CSV files.
type DirAdapter struct {
	path  string
	files map[string]*os.File
}

// NewDirAdapter returns an initialized DirAdapter
func NewDirAdapter(path string) *DirAdapter {
	return &DirAdapter{
		path:  path,
		files: map[string]*os.File{},
	}
}

// SHA1 returns an error.
func (adapter *DirAdapter) SHA1() (string, error) {
	return "", errors.New("cannot take SHA1 of directory")
}

// DirSHA1 returns the SHA1 of all the .txt files in the main directory, sorted, and concatenated.
func (adapter *DirAdapter) DirSHA1() (string, error) {
	f, err := os.Open(adapter.path)
	if err != nil {
		return "", err
	}
	fis, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return "", err
	}
	// Sort the files
	sort.Slice(fis, func(i, j int) bool { return fis[i].Name() < fis[j].Name() })
	// Generate SHA1
	h := sha1.New()
	for _, fi := range fis {
		fn := fi.Name()
		if fi.IsDir() || !strings.HasSuffix(fn, ".txt") || strings.HasPrefix(fn, ".") || strings.Contains(fn, "/") {
			continue
		}
		f, err := os.Open(filepath.Join(adapter.path, fi.Name()))
		if err != nil {
			return "", err
		}
		io.Copy(h, f)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// Open the adapter. Return an error if the directory does not exist.
func (adapter *DirAdapter) Open() error {
	if !adapter.Exists() {
		return errors.New("file does not exist")
	}
	return nil
}

// Close the adapter.
func (adapter *DirAdapter) Close() error {
	for _, f := range adapter.files {
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Path returns the directory path.
func (adapter *DirAdapter) Path() string {
	return adapter.path
}

// OpenFile opens a file in the directory. Returns an error if the file cannot be read.
func (adapter *DirAdapter) OpenFile(filename string, cb func(io.Reader)) error {
	in, err := os.Open(filepath.Join(adapter.path, filename))
	if err != nil {
		return err
	}
	defer in.Close()
	cb(in)
	return nil
}

// ReadRows opens the file and runs the callback for each row. An error is returned if the file cannot be read.
func (adapter *DirAdapter) ReadRows(filename string, cb func(Row)) error {
	return adapter.OpenFile(filename, func(in io.Reader) {
		ReadRows(in, cb)
	})
}

// Exists checks if the specified directory exists.
func (adapter *DirAdapter) Exists() bool {
	// Is the path a directory
	fi, err := os.Stat(adapter.path)
	if err != nil {
		return false
	}
	return fi.Mode().IsDir()
}

// WriteRows writes with only Flush at the end.
func (adapter *DirAdapter) WriteRows(filename string, rows [][]string) error {
	// Is this file open
	in, ok := adapter.files[filename]
	if !ok {
		i, err := os.Create(filepath.Join(adapter.path, filename))
		if err != nil {
			return err
		}
		in = i
		adapter.files[filename] = in
	}
	w := csv.NewWriter(in)
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	return nil
}

/////////////////////

// ZipWriterAdapter functions the same as DirAdapter, but writes to a temporary directory, and creates a zip archive when closed.
type ZipWriterAdapter struct {
	outpath string
	DirAdapter
}

// NewZipWriterAdapter returns a new ZipWriterAdapter.
func NewZipWriterAdapter(path string) *ZipWriterAdapter {
	tmpdir, err := ioutil.TempDir("", "gtfs")
	if err != nil {
		return nil
	}
	return &ZipWriterAdapter{
		outpath:    path,
		DirAdapter: *NewDirAdapter((tmpdir)),
	}
}

// Close creates a zip archive of all the written files at the specified destination.
func (adapter *ZipWriterAdapter) Close() error {
	out, err := os.Create(adapter.outpath)
	if err != nil {
		return nil
	}
	w := zip.NewWriter(out)
	defer w.Close()
	for k, f := range adapter.DirAdapter.files {
		// Seek to beginning of file
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return err
		}
		// Create zip name and copy to zip
		if wf, err := w.Create(k); err != nil {
			return err
		} else if _, err := io.Copy(wf, f); err != nil {
			return err
		}
		// Close and remove file
		if err := f.Close(); err != nil {
			return err
		} else if err := os.Remove(f.Name()); err != nil {
			return err
		}
	}
	if err := os.Remove(adapter.DirAdapter.path); err != nil {
		return err
	}
	return nil
}
