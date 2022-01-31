package tlcsv

import (
	"archive/zip"
	"crypto/sha1"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/interline-io/transitland-lib/internal/download"
	"github.com/interline-io/transitland-lib/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
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
	url    string
	secret download.Secret
	auth   tl.FeedAuthorization
	ZipAdapter
}

func (adapter *URLAdapter) SetAuth(auth tl.FeedAuthorization, secret download.Secret) {
	adapter.secret = secret
	adapter.auth = auth
}

// Open the adapter, and download the provided URL to a temporary file.
func (adapter *URLAdapter) Open() error {
	if adapter.ZipAdapter.path != "" {
		return nil // already open
	}
	// Remove and keep internal path prefix
	url := adapter.url
	fragment := ""
	split := strings.SplitN(adapter.url, "#", 2)
	if len(split) > 1 {
		url = split[0]
		fragment = split[1]
	}
	// Download to temporary file
	tmpfilepath, err := download.AuthenticatedRequest(url, adapter.secret, adapter.auth)
	if err != nil {
		return err
	}
	// Add internal path prefix back
	adapter.ZipAdapter = ZipAdapter{
		path:        tmpfilepath + "#" + fragment,
		tmpfilepath: tmpfilepath, // delete on close
	}
	return adapter.ZipAdapter.Open()
}

// Close the adapter, and remove the temporary file. An error is returned if the file could not be deleted.
func (adapter *URLAdapter) Close() error {
	return adapter.ZipAdapter.Close()
}

/////////////////////

// ZipAdapter supports reading from zip archives.
type ZipAdapter struct {
	path           string
	internalPrefix string
	tmpfilepath    string
}

// NewZipAdapter returns an initialized zip adapter.
func NewZipAdapter(path string) *ZipAdapter {
	return &ZipAdapter{path: path}
}

// Open the adapter. Return an error if the file does not exist.
func (adapter *ZipAdapter) Open() error {
	// Split fragment
	spliturl := strings.SplitN(adapter.path, "#", 2)
	if len(spliturl) > 1 {
		adapter.path = spliturl[0]
		adapter.internalPrefix = spliturl[1]
	}
	if !adapter.Exists() {
		return errors.New("file does not exist")
	}
	// Try to auto discover internal path fragment if unspecified
	if adapter.internalPrefix == "" {
		pfx, err := adapter.findInternalPrefix()
		if err != nil {
			return err
		}
		log.Debug("Using auto-discovered internal prefix: %s", pfx)
		adapter.internalPrefix = pfx
	} else if strings.HasSuffix(adapter.internalPrefix, ".zip") {
		// If the internal prefix is a zip, extract this to a temp file
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
			return err
		}
		adapter.path = tmpfilepath
		adapter.tmpfilepath = tmpfilepath
		adapter.internalPrefix = ""
	}
	return nil
}

// Close the adapter.
func (adapter *ZipAdapter) Close() error {
	if adapter.tmpfilepath != "" {
		log.Debug("removing temp file: %s", adapter.tmpfilepath)
		if err := os.Remove(adapter.tmpfilepath); err != nil {
			return err
		}
	}
	return nil
}

// Path returns the path to the zip file.
func (adapter *ZipAdapter) Path() string {
	return adapter.path
}

// Exists returns if the zip file exists.
func (adapter *ZipAdapter) Exists() bool {
	// Is the file readable
	r, err := zip.OpenReader(adapter.path)
	if err != nil {
		return false
	}
	r.Close()
	return true
}

// OpenFile opens the file inside the archive and passes it to the provided callback.
func (adapter *ZipAdapter) OpenFile(filename string, cb func(io.Reader)) error {
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
	if err != nil {
		return err
	}
	defer in.Close()
	cb(in)
	return nil
}

// ReadRows opens the specified file and runs the callback on each Row. An error is returned if the file cannot be read.
func (adapter *ZipAdapter) ReadRows(filename string, cb func(Row)) error {
	return adapter.OpenFile(filename, func(in io.Reader) {
		ReadRows(in, cb)
	})
}

// SHA1 returns the SHA1 checksum of the zip archive.
func (adapter *ZipAdapter) SHA1() (string, error) {
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
func (adapter *ZipAdapter) DirSHA1() (string, error) {
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
		if err != nil {
			return "", err
		}
		defer f.Close()
		io.Copy(h, f)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// FileInfos returns a list of os.FileInfo for all .txt files.
func (adapter *ZipAdapter) FileInfos() ([]os.FileInfo, error) {
	ret := []os.FileInfo{}
	r, err := zip.OpenReader(adapter.path)
	if err != nil {
		return ret, err
	}
	defer r.Close()
	sort.Slice(r.File, func(i, j int) bool { return r.File[i].Name < r.File[j].Name })
	for _, zf := range r.File {
		fi := zf.FileInfo()
		fn := zf.Name
		if adapter.internalPrefix != "" {
			fn = strings.Replace(zf.Name, adapter.internalPrefix+"/", "", 1) // remove internalPrefix
		}
		if fi.IsDir() || !strings.HasSuffix(fn, ".txt") || strings.HasPrefix(fn, ".") || strings.Contains(fn, "/") {
			continue
		}
		ret = append(ret, fi)
	}
	return ret, nil
}

func (adapter *ZipAdapter) findInternalPrefix() (string, error) {
	r, err := zip.OpenReader(adapter.path)
	if err != nil {
		return "", err
	}
	prefixes := []string{}
	defer r.Close()
	for _, zf := range r.File {
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
		return prefixes[0], nil
	}
	return "", nil
}

/////////////////////

// DirAdapter supports plain directories of CSV files.
type DirAdapter struct {
	path  string
	files map[string]*os.File
}

// NewDirAdapter returns an initialized DirAdapter.
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
	h := sha1.New()
	fis, err := adapter.FileInfos()
	if err != nil {
		return "", err
	}
	for _, fi := range fis {
		f, err := os.Open(filepath.Join(adapter.path, fi.Name()))
		if err != nil {
			return "", err
		}
		io.Copy(h, f)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// FileInfos returns a list of os.FileInfo for all top-level .txt files.
func (adapter *DirAdapter) FileInfos() ([]os.FileInfo, error) {
	ret := []os.FileInfo{}
	f, err := os.Open(adapter.path)
	if err != nil {
		return ret, err
	}
	fis, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return ret, err
	}
	// Sort the files
	sort.Slice(fis, func(i, j int) bool { return fis[i].Name() < fis[j].Name() })
	// Generate SHA1
	for _, fi := range fis {
		fn := fi.Name()
		if fi.IsDir() || !strings.HasSuffix(fn, ".txt") || strings.HasPrefix(fn, ".") || strings.Contains(fn, "/") {
			continue
		}
		if err != nil {
			return ret, err
		}
		ret = append(ret, fi)
	}
	return ret, nil
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

// AddFile directly adds a file to this directory. Useful for manual feed operations.
func (adapter *DirAdapter) AddFile(filename string, reader io.Reader) error {
	in, ok := adapter.files[filename]
	if !ok {
		i, err := os.Create(filepath.Join(adapter.path, filename))
		if err != nil {
			return err
		}
		in = i
		adapter.files[filename] = in
	}
	if _, err := io.Copy(in, reader); err != nil {
		return err
	}
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
