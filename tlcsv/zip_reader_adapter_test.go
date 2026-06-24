package tlcsv

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testpath"
)

func TestZipReaderAdapter(t *testing.T) {
	b, err := os.ReadFile(testpath.RelPath("testdata/gtfs-examples/example.zip"))
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewZipReaderAdapter(bytes.NewReader(b), int64(len(b)))
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
	adapter, err := NewZipReaderAdapter(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
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

	// FileInfos reports the feed files by base name (prefix stripped), so file-info
	// stats and OpenFile(fi.Name()) agree even when the feed is in a subdirectory.
	fis, err := adapter.FileInfos()
	if err != nil {
		t.Fatalf("FileInfos: %v", err)
	}
	var names []string
	for _, fi := range fis {
		names = append(names, fi.Name())
		if err := adapter.OpenFile(fi.Name(), func(io.Reader) {}); err != nil {
			t.Errorf("OpenFile(%q) from FileInfos: %v", fi.Name(), err)
		}
	}
	want := []string{"agency.txt", "routes.txt", "stops.txt"}
	if len(names) != len(want) {
		t.Fatalf("FileInfos names = %v, want %v", names, want)
	}
	for i, n := range want {
		if names[i] != n {
			t.Errorf("FileInfos[%d] = %q, want %q (got %v)", i, names[i], n, names)
		}
	}
}

func TestZipReaderAdapter_ExplicitPrefix(t *testing.T) {
	// Two complete feeds in separate directories: auto-discovery is ambiguous, so an
	// explicit prefix is the only way to select one.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range []string{
		"feedA/agency.txt", "feedA/stops.txt",
		"feedB/agency.txt", "feedB/stops.txt",
	} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		// Tag each file with its directory so we can tell which feed we read.
		if _, err := w.Write([]byte("id\n" + name + "\n")); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	b := buf.Bytes()

	// Auto-discovery fails on the ambiguous archive.
	auto, err := NewZipReaderAdapter(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	if err := auto.Open(); err == nil {
		t.Error("Open() should fail on an ambiguous archive without an explicit prefix")
	}

	// An explicit prefix selects feedB and reads its files under that root.
	adapter, err := NewZipReaderAdapterWithPrefix(bytes.NewReader(b), int64(len(b)), "feedB")
	if err != nil {
		t.Fatal(err)
	}
	if err := adapter.Open(); err != nil {
		t.Fatalf("Open() with explicit prefix: %v", err)
	}
	got := ""
	if err := adapter.OpenFile("stops.txt", func(r io.Reader) {
		bs, _ := io.ReadAll(r)
		got = string(bs)
	}); err != nil {
		t.Fatalf("OpenFile(stops.txt): %v", err)
	}
	if !strings.Contains(got, "feedB/stops.txt") {
		t.Errorf("read %q, want the feedB copy", got)
	}
}
