package tlcsv

import (
	"archive/zip"
	"context"
	"crypto/sha1"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/internal/tags"
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

/////////////////////

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

/////////////////////

// URLAdapter downloads a GTFS URL to a temporary file, and removes the file when it is closed.
type URLAdapter struct {
	url     string
	reqOpts []request.RequestOption
	ZipAdapter
}

func NewURLAdapter(address string, opts ...request.RequestOption) *URLAdapter {
	return &URLAdapter{
		url:     address,
		reqOpts: opts,
	}
}

func (adapter *URLAdapter) String() string {
	return adapter.url
}

// Open the adapter, and download the provided URL to a temporary file.
func (adapter *URLAdapter) Open() error {
	if adapter.ZipAdapter.path != "" {
		return nil // already open
	}
	// Remove and keep internal path prefix
	url, fragment, _ := strings.Cut(adapter.url, "#")
	// Download to temporary file
	tmpfile, fr, err := request.AuthenticatedRequestDownload(context.TODO(), url, adapter.reqOpts...)
	if err != nil {
		return err
	}
	if fr.FetchError != nil {
		return fr.FetchError
	}
	// Add internal path prefix back
	adapter.ZipAdapter = ZipAdapter{
		path:           tmpfile,
		internalPrefix: fragment,
		tmpfiles:       []string{tmpfile},
	}
	return adapter.ZipAdapter.Open()
}

///////////////

// Temporary zip adapter

func NewTmpZipAdapterFromReader(reader io.Reader, fragment string) (*ZipAdapter, error) {
	// Create temp file
	tmpfile, err := os.CreateTemp("", "gtfs")
	if err != nil {
		return nil, err
	}
	defer tmpfile.Close()

	// Read stream to a temporary file
	if _, err := io.Copy(tmpfile, reader); err != nil {
		return nil, err
	}
	// Add internal path prefix back
	adapter := ZipAdapter{
		path:           tmpfile.Name(),
		internalPrefix: fragment,
		tmpfiles:       []string{tmpfile.Name()},
	}
	return &adapter, nil
}

/////////////////////

// ZipAdapter supports reading from zip archives.
type ZipAdapter struct {
	path           string
	internalPrefix string
	tmpfiles       []string
}

// NewZipAdapter returns an initialized zip adapter.
func NewZipAdapter(path string) *ZipAdapter {
	return &ZipAdapter{path: strings.TrimPrefix(path, "file://")}
}

func (adapter *ZipAdapter) String() string {
	return adapter.path
}

// Open the adapter. Return an error if the file does not exist.
func (adapter *ZipAdapter) Open() error {
	// Split fragment
	if path, prefix, ok := strings.Cut(adapter.path, "#"); ok {
		adapter.path = path
		adapter.internalPrefix = prefix
	}
	if !adapter.Exists() {
		return errors.New("file does not exist or invalid data")
	}
	// Try to auto discover internal path fragment if unspecified
	if adapter.internalPrefix == "" {
		pfx, err := adapter.findInternalPrefix()
		if err != nil {
			return err
		}
		adapter.internalPrefix = pfx
	} else if strings.HasSuffix(adapter.internalPrefix, ".zip") {
		// If the internal prefix is a zip, extract this to a temp file
		pf := adapter.internalPrefix
		adapter.internalPrefix = ""
		tmpfilepath := ""
		err := adapter.OpenFile(pf, func(r io.Reader) {
			// Create the file
			tmpfile, _ := os.CreateTemp("", "gtfs-nested")
			defer tmpfile.Close()
			// Get the full path
			tmpfilepath = tmpfile.Name()
			// Add to temp file list
			adapter.tmpfiles = append(adapter.tmpfiles, tmpfilepath)
			// Write the body to file
			log.For(context.TODO()).Debug().Str("dst", tmpfilepath).Str("src", adapter.path).Str("prefix", pf).Msg("zip adapter: extracted internal zip")
			io.Copy(tmpfile, r)
		})
		if err != nil {
			return err
		}
		adapter.path = tmpfilepath
		adapter.internalPrefix = ""
	}
	if adapter.internalPrefix != "" {
		log.For(context.TODO()).Trace().Msgf("zip adapter: using internal prefix: %s", adapter.internalPrefix)
	}
	return nil
}

// Close the adapter.
func (adapter *ZipAdapter) Close() error {
	for _, tmpfile := range adapter.tmpfiles {
		log.For(context.TODO()).Debug().Msgf("zip adapter: removing temp file: %s", tmpfile)
		if err := os.Remove(tmpfile); err != nil {
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
	defer f.Close()
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
		// Ignore directories, subdirs, dot files
		if fi.IsDir() || strings.HasPrefix(fn, ".") || strings.Contains(fn, "/") {
			continue
		}
		// Only generate stats for files with lowercase names that end with .txt
		if fi.Name() != strings.ToLower(fi.Name()) || !strings.HasSuffix(fi.Name(), ".txt") {
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
		if fi.IsDir() || strings.HasPrefix(fn, ".") || strings.Contains(fn, "/") {
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
		pfx := prefixes[0]
		if pfx == "." {
			pfx = ""
		}
		return pfx, nil
	}
	return "", nil
}

/////////////////////

// sortColumnRegistrar receives sort metadata from a Writer at file-write
// time. ZipWriterAdapter inherits the implementation via embedded DirAdapter.
type sortColumnRegistrar interface {
	registerSortColumns(filename string, cols []*tags.FieldInfo)
}

// DirAdapter supports plain directories of CSV files.
type DirAdapter struct {
	path              string
	files             map[string]*os.File
	geojsonFeatures   map[string][]*geojson.Feature
	sortOptions       adapters.StandardizedSortOptions
	sortColumnsByFile map[string][]*tags.FieldInfo
}

// NewDirAdapter returns an initialized DirAdapter.
func NewDirAdapter(path string) *DirAdapter {
	return &DirAdapter{
		path:              strings.TrimPrefix(path, "file://"),
		files:             map[string]*os.File{},
		geojsonFeatures:   map[string][]*geojson.Feature{},
		sortColumnsByFile: map[string][]*tags.FieldInfo{},
	}
}

func (adapter *DirAdapter) registerSortColumns(filename string, cols []*tags.FieldInfo) {
	adapter.sortColumnsByFile[filename] = cols
}

// String
func (adapter *DirAdapter) String() string {
	return adapter.path
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
		// Only generate stats for files with lowercase names that end with .txt
		if fi.Name() != strings.ToLower(fi.Name()) || !strings.HasSuffix(fi.Name(), ".txt") {
			continue
		}
		f, err := os.Open(filepath.Join(adapter.path, fi.Name()))
		if err != nil {
			return "", err
		}
		io.Copy(h, f)
		f.Close()
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
		if fi.IsDir() || strings.HasPrefix(fn, ".") || strings.Contains(fn, "/") {
			continue
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

func (adapter *DirAdapter) SetStandardizedSortOptions(opts adapters.StandardizedSortOptions) {
	adapter.sortOptions = opts
}

// resolveSortColumns returns the user-supplied override if set, otherwise
// the captured per-file defaults; nil signals "skip this file".
func (adapter *DirAdapter) resolveSortColumns(filename string) []*tags.FieldInfo {
	captured := adapter.sortColumnsByFile[filename]
	if len(adapter.sortOptions.SortColumns) == 0 {
		return captured
	}
	// User-supplied column names still get type-aware sorting if we recognize them.
	kindByName := map[string]tags.SortKind{}
	for _, c := range captured {
		kindByName[c.Name] = c.Kind
	}
	out := make([]*tags.FieldInfo, 0, len(adapter.sortOptions.SortColumns))
	for i, name := range adapter.sortOptions.SortColumns {
		out = append(out, &tags.FieldInfo{Name: name, Kind: kindByName[name], SortOrder: i + 1})
	}
	return out
}

// StandardizedSortCSVFiles sorts every registered .txt file in place.
//
// TODO: this loads each file fully into memory (csv.ReadAll + in-memory
// sort.SliceStable + truncate-and-rewrite). For large feeds, files like
// stop_times.txt and shapes.txt can be tens of millions of rows and will
// drive RSS up accordingly. The feature is opt-in, so the cost is only
// paid when callers ask for it. A streaming external-merge replacement is
// possible (write sorted runs to temp files, then k-way merge), but the
// same memory pressure is reachable through other endpoints, so a
// caller-bounded fix here can be defeated by an attacker who simply hits
// a different code path. Revisit alongside any general resource-limit
// work, not as a one-off.
func (adapter *DirAdapter) StandardizedSortCSVFiles() error {
	sortOrder := adapter.sortOptions.ApplySort
	if sortOrder == "" {
		return nil
	}
	descending := sortOrder == adapters.SortDesc

	type plan struct {
		name string
		cols []*tags.FieldInfo
	}
	var plans []plan
	for filename := range adapter.files {
		if !strings.HasSuffix(filename, ".txt") {
			continue
		}
		cols := adapter.resolveSortColumns(filename)
		if len(cols) == 0 {
			continue
		}
		plans = append(plans, plan{name: filename, cols: cols})
	}

	for _, p := range plans {
		f := adapter.files[p.name]
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("failed to seek %s: %w", p.name, err)
		}
		allRows, err := csv.NewReader(f).ReadAll()
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", p.name, err)
		}
		if len(allRows) == 0 {
			continue
		}
		header, dataRows := allRows[0], allRows[1:]
		if keys := resolveHeaderKeys(header, p.cols); len(keys) > 0 {
			sortRows(dataRows, keys, descending)
		}

		if err := f.Truncate(0); err != nil {
			return fmt.Errorf("failed to truncate %s: %w", p.name, err)
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("failed to seek %s: %w", p.name, err)
		}
		w := csv.NewWriter(f)
		if err := w.Write(header); err != nil {
			return fmt.Errorf("failed to write header to %s: %w", p.name, err)
		}
		if err := w.WriteAll(dataRows); err != nil {
			return fmt.Errorf("failed to write rows to %s: %w", p.name, err)
		}
		w.Flush()
		if err := w.Error(); err != nil {
			return fmt.Errorf("failed to flush writer for %s: %w", p.name, err)
		}
	}

	return nil
}

// Close the adapter. Flushes any buffered GeoJSON files before closing.
func (adapter *DirAdapter) Close() error {
	// Sort files if requested
	if adapter.sortOptions.ApplySort != "" {
		if err := adapter.StandardizedSortCSVFiles(); err != nil {
			return err
		}
	}

	// Flush all buffered GeoJSON files
	for filename, features := range adapter.geojsonFeatures {
		if len(features) > 0 {
			if err := adapter.flushGeoJSON(filename, features); err != nil {
				return err
			}
		}
	}
	adapter.geojsonFeatures = map[string][]*geojson.Feature{}

	return adapter.CloseFiles()
}

// CloseFiles closes all open file handles.
func (adapter *DirAdapter) CloseFiles() error {
	for _, f := range adapter.files {
		if err := f.Close(); err != nil {
			return err
		}
	}
	adapter.files = map[string]*os.File{}
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

// WriteFeatures buffers GeoJSON features to be written when the adapter is closed.
func (adapter *DirAdapter) WriteFeatures(filename string, features []*geojson.Feature) error {
	if len(features) == 0 {
		return nil
	}
	adapter.geojsonFeatures[filename] = append(adapter.geojsonFeatures[filename], features...)
	return nil
}

// flushGeoJSON writes all buffered features for a file as a FeatureCollection.
func (adapter *DirAdapter) flushGeoJSON(filename string, features []*geojson.Feature) error {
	// Close existing file if open
	if in, ok := adapter.files[filename]; ok {
		in.Close()
		delete(adapter.files, filename)
	}

	// Create new file
	in, err := os.Create(filepath.Join(adapter.path, filename))
	if err != nil {
		return err
	}
	adapter.files[filename] = in

	// Write FeatureCollection
	fc := geojson.FeatureCollection{
		Features: features,
	}
	encoder := json.NewEncoder(in)
	encoder.SetIndent("", "  ")
	return encoder.Encode(&fc)
}

/////////////////////

// ZipWriterAdapter functions the same as DirAdapter, but writes to a temporary directory, and creates a zip archive when closed.
type ZipWriterAdapter struct {
	outpath string
	DirAdapter
}

// NewZipWriterAdapter returns a new ZipWriterAdapter.
func NewZipWriterAdapter(path string) *ZipWriterAdapter {
	tmpdir, err := os.MkdirTemp("", "gtfs")
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
	// Sort files if requested
	if adapter.sortOptions.ApplySort != "" {
		if err := adapter.DirAdapter.StandardizedSortCSVFiles(); err != nil {
			return err
		}
	}

	// Flush any buffered GeoJSON files first
	for filename, features := range adapter.DirAdapter.geojsonFeatures {
		if len(features) > 0 {
			if err := adapter.DirAdapter.flushGeoJSON(filename, features); err != nil {
				return err
			}
		}
	}
	adapter.DirAdapter.geojsonFeatures = map[string][]*geojson.Feature{}

	out, err := os.Create(adapter.outpath)
	if err != nil {
		return err
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
