package tlcsv

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/interline-io/log"
)

// NewTmpZipAdapterFromReader copies a stream to a temporary zip file and returns a
// ZipAdapter over it (removed on Close). An optional fragment selects an internal
// feed root within the archive.
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

// ZipAdapter reads a GTFS zip archive on disk. It is a thin disk-backed shell over
// ZipReaderAdapter: every read and the feed-root detection are delegated to a
// short-lived ZipReaderAdapter opened over the file (so it stays stateless and
// concurrency-safe, exactly like before). The only things it adds on top are the
// two filesystem-only cases the in-memory adapter can't do — a path#fragment
// selecting an internal prefix, and a fragment that is itself a nested .zip, which
// is extracted to a temp file and read in turn.
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

// Open resolves the archive's location and feed-root prefix once. It splits a
// path#fragment, extracts a nested .zip fragment to a temp file, and then caches
// the (auto-discovered when unspecified) internal prefix so subsequent reads apply
// it without re-discovering.
func (adapter *ZipAdapter) Open() error {
	// A path#fragment selects an internal prefix (a feed root, or a nested .zip).
	if path, prefix, ok := strings.Cut(adapter.path, "#"); ok {
		adapter.path = path
		adapter.internalPrefix = prefix
	}
	// A fragment that is itself a zip: extract it to a temp file and read that.
	if strings.HasSuffix(adapter.internalPrefix, ".zip") {
		extracted, err := adapter.extractNestedZip(adapter.internalPrefix)
		if err != nil {
			return err
		}
		adapter.path = extracted
		adapter.internalPrefix = ""
	}
	// Resolve + cache the feed root via the inner adapter (auto-discovers when the
	// prefix is unset; errors on a missing/ambiguous one).
	inner, file, err := adapter.reader()
	if err != nil {
		return err
	}
	defer file.Close()
	if err := inner.Open(); err != nil {
		return err
	}
	adapter.internalPrefix = inner.internalPrefix
	if adapter.internalPrefix != "" {
		log.For(context.TODO()).Trace().Msgf("zip adapter: using internal prefix: %s", adapter.internalPrefix)
	}
	return nil
}

// reader opens a ZipReaderAdapter over the archive for a single operation, with the
// adapter's (resolved) internal prefix applied. The caller closes the returned file.
func (adapter *ZipAdapter) reader() (*ZipReaderAdapter, *os.File, error) {
	return openFileZipReader(adapter.path, adapter.internalPrefix)
}

// openFileZipReader opens a zip file on disk and wraps it in a ZipReaderAdapter with
// the given internal prefix (empty = auto-discover on Open), reading lazily through
// the returned open file (which the caller must close).
func openFileZipReader(path, internalPrefix string) (*ZipReaderAdapter, *os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	inner, err := NewZipReaderAdapterWithPrefix(f, fi.Size(), internalPrefix)
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	return inner, f, nil
}

// extractNestedZip copies the nested .zip at nestedPath out of the archive into a
// temp file (removed on Close) and returns the temp file's path.
func (adapter *ZipAdapter) extractNestedZip(nestedPath string) (string, error) {
	// Read the nested entry by its full path, so the outer archive must have no
	// prefix of its own applied.
	outer, file, err := openFileZipReader(adapter.path, "")
	if err != nil {
		return "", err
	}
	defer file.Close()
	tmpfile, err := os.CreateTemp("", "gtfs-nested")
	if err != nil {
		return "", err
	}
	defer tmpfile.Close()
	adapter.tmpfiles = append(adapter.tmpfiles, tmpfile.Name())
	log.For(context.TODO()).Debug().Str("dst", tmpfile.Name()).Str("src", adapter.path).Str("prefix", nestedPath).Msg("zip adapter: extracting internal zip")
	var copyErr error
	if err := outer.OpenFile(nestedPath, func(r io.Reader) { _, copyErr = io.Copy(tmpfile, r) }); err != nil {
		return "", err
	}
	if copyErr != nil {
		return "", copyErr
	}
	return tmpfile.Name(), nil
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

// Exists returns if the zip file exists and is a readable archive.
func (adapter *ZipAdapter) Exists() bool {
	inner, file, err := adapter.reader()
	if err != nil {
		return false
	}
	defer file.Close()
	return inner.Exists()
}

// OpenFile opens the file inside the archive and passes it to the provided callback.
func (adapter *ZipAdapter) OpenFile(filename string, cb func(io.Reader)) error {
	inner, file, err := adapter.reader()
	if err != nil {
		return err
	}
	defer file.Close()
	return inner.OpenFile(filename, cb)
}

// ReadRows opens the specified file and runs the callback on each Row. An error is returned if the file cannot be read.
func (adapter *ZipAdapter) ReadRows(filename string, cb func(Row)) error {
	t0 := time.Now()
	log.For(context.TODO()).Trace().Str("filename", filename).Msg("tlcsv: read pass")
	err := adapter.OpenFile(filename, func(in io.Reader) {
		ReadRows(in, cb)
	})
	log.For(context.TODO()).Trace().Str("filename", filename).Int("elapsed_ms", int(time.Since(t0).Milliseconds())).Msg("tlcsv: read pass complete")
	return err
}

// SHA1 returns the SHA1 checksum of the zip archive.
func (adapter *ZipAdapter) SHA1() (string, error) {
	inner, file, err := adapter.reader()
	if err != nil {
		return "", err
	}
	defer file.Close()
	return inner.SHA1()
}

// DirSHA1 returns the SHA1 of all the .txt files in the main directory, sorted, and concatenated.
func (adapter *ZipAdapter) DirSHA1() (string, error) {
	inner, file, err := adapter.reader()
	if err != nil {
		return "", err
	}
	defer file.Close()
	return inner.DirSHA1()
}

// FileInfos returns a list of os.FileInfo for all top-level .txt files.
func (adapter *ZipAdapter) FileInfos() ([]os.FileInfo, error) {
	inner, file, err := adapter.reader()
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return inner.FileInfos()
}

func (adapter *ZipAdapter) findInternalPrefix() (string, error) {
	inner, file, err := openFileZipReader(adapter.path, "")
	if err != nil {
		return "", err
	}
	defer file.Close()
	return detectInternalPrefix(inner)
}
