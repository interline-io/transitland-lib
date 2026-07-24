package testutil

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// ZipDirToTemp writes every regular file in dir into a new temp .zip and returns
// its path. Feed fixtures are stored as directories so they stay the single source
// of truth; tests that need a zip (fetch, import) zip them on the fly.
func ZipDirToTemp(t *testing.T, dir string) string {
	t.Helper()
	tmp, err := os.CreateTemp("", "gtfs-*.zip")
	if err != nil {
		t.Fatal(err)
	}
	defer tmp.Close()
	zw := zip.NewWriter(tmp)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		buf, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		f, err := zw.Create(e.Name())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(buf); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return tmp.Name()
}
