package tlcsv

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testpath"
)

func TestZipReaderAdapter(t *testing.T) {
	b, err := os.ReadFile(testpath.RelPath("testdata/gtfs-examples/example.zip"))
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewZipReaderAdapterFromBytes(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := adapter.Open(); err != nil {
		t.Fatal(err)
	}
	if !adapter.Exists() {
		t.Fatal("Exists() = false")
	}
	// Parity with the path-based ZipAdapter: the known whole-archive and per-dir
	// checksums for example.zip.
	sha1, err := adapter.SHA1()
	if err != nil {
		t.Fatalf("SHA1: %v", err)
	}
	if sha1 != "ce0a38dd6d4cfdac6aebe003181b6b915390a3b8" {
		t.Errorf("SHA1 = %s, want ce0a38dd...", sha1)
	}
	dirSHA1, err := adapter.DirSHA1()
	if err != nil {
		t.Fatalf("DirSHA1: %v", err)
	}
	if dirSHA1 != "7a5c69b5466746213eb3cb6d907a7004073eca4d" {
		t.Errorf("DirSHA1 = %s, want 7a5c69b5...", dirSHA1)
	}
	reader, err := NewReaderFromAdapter(adapter)
	if err != nil {
		t.Fatal(err)
	}
	if errs := reader.ValidateStructure(); len(errs) > 0 {
		t.Errorf("ValidateStructure: %v", errs)
	}
	rows := 0
	if err := adapter.ReadRows("stops.txt", func(Row) { rows++ }); err != nil {
		t.Fatal(err)
	}
	if rows == 0 {
		t.Error("stops.txt streamed 0 rows")
	}
}

func TestZipReaderAdapter_NestedPrefix(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range []string{"feed/agency.txt", "feed/stops.txt", "feed/routes.txt"} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte("id\nx\n")); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	adapter, err := NewZipReaderAdapterFromBytes(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if err := adapter.Open(); err != nil {
		t.Fatal(err)
	}
	// The feed root "feed/" is auto-discovered, so flat names resolve under it.
	if err := adapter.OpenFile("stops.txt", func(io.Reader) {}); err != nil {
		t.Errorf("OpenFile(stops.txt) under nested prefix: %v", err)
	}
	if err := adapter.OpenFile("missing.txt", func(io.Reader) {}); err == nil {
		t.Error("OpenFile(missing.txt) should error")
	}
}
