package tlcsv

import (
	"archive/zip"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/internal/tags"
	"github.com/twpayne/go-geom/encoding/geojson"
)

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
	fis, err := adapter.FileInfos()
	if err != nil {
		return "", err
	}
	names := make([]string, len(fis))
	for i, fi := range fis {
		names[i] = fi.Name()
	}
	return dirSHA1(names, adapter.OpenFile)
}

// FileInfos returns a list of os.FileInfo for all top-level feed files.
func (adapter *DirAdapter) FileInfos() ([]os.FileInfo, error) {
	return feedFileInfos(adapter, "")
}

// Walk yields the directory's top-level entries (directories suffixed with "/").
// The directory adapter is flat, so it does not recurse — prefix selects a
// subdirectory to list. Read-only; unrelated to Open.
func (adapter *DirAdapter) Walk(prefix string) iter.Seq2[WalkFile, error] {
	return func(yield func(WalkFile, error) bool) {
		entries, err := os.ReadDir(filepath.Join(adapter.path, prefix))
		if err != nil {
			yield(WalkFile{}, err)
			return
		}
		for _, e := range entries {
			fi, err := e.Info()
			if err != nil {
				if !yield(WalkFile{}, err) {
					return
				}
				continue
			}
			name := e.Name()
			if e.IsDir() {
				name += "/"
			}
			if !yield(WalkFile{Path: name, FileInfo: fi}, nil) {
				return
			}
		}
	}
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
	t0 := time.Now()
	log.For(context.TODO()).Trace().Str("filename", filename).Msg("tlcsv: read pass")
	err := adapter.OpenFile(filename, func(in io.Reader) {
		ReadRows(in, cb)
	})
	log.For(context.TODO()).Trace().Str("filename", filename).Int("elapsed_ms", int(time.Since(t0).Milliseconds())).Msg("tlcsv: read pass complete")
	return err
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
