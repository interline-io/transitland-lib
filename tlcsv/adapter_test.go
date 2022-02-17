package tlcsv

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tl"
)

func getTestAdapters() map[string]func() Adapter {
	adapters := map[string]func() Adapter{
		"DirAdapter":          func() Adapter { return NewDirAdapter(testutil.ExampleDir.URL) },
		"ZipAdapter":          func() Adapter { return NewZipAdapter(testutil.ExampleZip.URL) },
		"ZipAdapterNestedDir": func() Adapter { return NewZipAdapter(testutil.ExampleZipNestedDir.URL) },
		"ZipAdapterNestedZip": func() Adapter { return NewZipAdapter(testutil.ExampleZipNestedZip.URL) },
		"OverlayAdapter":      func() Adapter { return NewOverlayAdapter(testutil.ExampleDir.URL) },
	}
	return adapters
}

func TestDirAdapter(t *testing.T) {
	v, ok := getTestAdapters()["DirAdapter"]
	if !ok {
		t.Error("no DirAdapter")
	}
	testAdapter(t, v())
	t.Run("DirSHA1", func(t *testing.T) {
		adapter, ok := v().(*DirAdapter)
		if !ok {
			t.Error("not DirAdapter!")
			return
		}
		s, err := adapter.DirSHA1()
		if err != nil {
			t.Error(err)
		}
		if s != testutil.ExampleDir.DirSHA1 {
			t.Errorf("got %s expect %s", s, testutil.ExampleDir.DirSHA1)
		}
	})
}

func TestOverlayAdapter(t *testing.T) {
	v, ok := getTestAdapters()["OverlayAdapter"]
	if !ok {
		t.Error("no OverlayAdapter")
	}
	testAdapter(t, v())
}

func TestZipAdapter(t *testing.T) {
	v, ok := getTestAdapters()["ZipAdapter"]
	if !ok {
		t.Error("no ZipAdapter")
	}
	testAdapter(t, v())
	t.Run("SHA1", func(t *testing.T) {
		adapter, ok := v().(*ZipAdapter)
		if !ok {
			t.Error("not ZipAdapter!")
			return
		}
		s, err := adapter.SHA1()
		if err != nil {
			t.Error(err)
		}
		if s != testutil.ExampleZip.SHA1 {
			t.Errorf("got %s expect %s", s, testutil.ExampleZip.SHA1)
		}
	})
	t.Run("DirSHA1", func(t *testing.T) {
		adapter, ok := v().(*ZipAdapter)
		if !ok {
			t.Error("not ZipAdapter!")
			return
		}
		s, err := adapter.DirSHA1()
		if err != nil {
			t.Error(err)
		}
		if s != testutil.ExampleZip.DirSHA1 {
			t.Errorf("got %s expect %s", s, testutil.ExampleZip.DirSHA1)
		}
	})
}

func TestZipAdapterNestedDir(t *testing.T) {
	v, ok := getTestAdapters()["ZipAdapterNestedDir"]
	if !ok {
		t.Error("no ZipAdapter with nested file")
		t.FailNow()
	}
	testAdapter(t, v())
}

func TestZipAdapter_findInternalPrefix(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		v := ZipAdapter{path: testutil.RelPath("test/data/example-nested-dir.zip")}
		if err := v.Open(); err != nil {
			t.Error(err)
			return
		}
		p, err := v.findInternalPrefix()
		if err != nil {
			t.Error(err)
		}
		expect := "example-nested-dir/example"
		if p != expect {
			t.Errorf("got '%s' expect '%s'", p, expect)
		}
	})
	t.Run("ambiguous", func(t *testing.T) {
		v := ZipAdapter{path: testutil.RelPath("test/data/example-nested-dir-ambiguous.zip")}
		v.internalPrefix = "example-nested-dir/example" // override for test
		if err := v.Open(); err != nil {
			t.Error(err)
			return
		}
		p, err := v.findInternalPrefix()
		if err == nil {
			t.Errorf("expected error for ambiguous prefixes")
		}
		expect := ""
		if p != expect {
			t.Errorf("got '%s' expect '%s'", p, expect)
		}
	})
}

func TestZipAdapterNestedZip(t *testing.T) {
	v, ok := getTestAdapters()["ZipAdapterNestedZip"]
	if !ok {
		t.Error("no ZipAdapter with nested zip")
		t.FailNow()
	}
	testAdapter(t, v())
}

func TestURLAdapter(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile(testutil.ExampleZip.URL)
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))
	defer ts.Close()
	// Main tests
	testAdapter(t, &URLAdapter{url: ts.URL})
	//
	t.Run("Download", func(t *testing.T) {
		a := URLAdapter{url: ts.URL}
		if err := a.Open(); err != nil {
			t.Error(err)
		}
		p := a.Path()
		if _, err := os.Stat(p); err != nil {
			t.Error("did not download file")
		}
		if err := a.Close(); err != nil {
			t.Error(err)
		}
		if _, err := os.Stat(p); err == nil {
			t.Error("did not remove temp file")
		}
	})
}

func TestZipWriterAdapter(t *testing.T) {
	// Perform various tests of the ZipWriterAdapter:
	// creates temporary shadow directory
	// removes temporary shadow directory
	// creates zip file when closed
	outf, err := ioutil.TempFile("", "zip")
	outpath := outf.Name()
	defer os.Remove(outpath)
	if err != nil {
		t.Error(err)
	}
	adapter := NewZipWriterAdapter(outpath)
	// Header
	if err := adapter.WriteRows("hello.txt", [][]string{{"one", "two", "three"}}); err != nil {
		t.Error(err)
	}
	// Body
	if err := adapter.WriteRows("hello.txt", [][]string{{"1", "2", "3"}}); err != nil {
		t.Error(err)
	}
	// Create Zip
	if err := adapter.Close(); err != nil {
		t.Error(err)
	}
	// Check that no temp files exist
	if _, err := os.Stat(adapter.path); !os.IsNotExist(err) {
		t.Errorf("expected temporary directory '%s' to have been removed", adapter.path)
	}
	// Read zip
	reader := ZipAdapter{path: outpath}
	if !reader.Exists() {
		t.Error("outpath does not exist")
	}
	reader.Open()
	defer reader.Close()
	rows := [][]string{}
	reader.ReadRows("hello.txt", func(row Row) {
		rows = append(rows, row.Row)
	})
	if len(rows) != 1 {
		t.Errorf("got %d rows, expected %d", len(rows), 1)
	} else {
		r := rows[0]
		if r[0] != "1" {
			t.Errorf("got %s expect %s", r[0], "1")
		}
	}
}

// Adapter interface tests
func testAdapter(t *testing.T, adapter Adapter) {
	openerr := adapter.Open()
	t.Run("Open", func(t *testing.T) {
		if openerr != nil {
			t.Error(openerr)
		}
	})
	t.Run("Exists", func(t *testing.T) {
		// TODO: doesnt check false cases
		if !adapter.Exists() {
			t.Errorf("got %t expected %t", false, true)
		}
	})
	t.Run("OpenFile", func(t *testing.T) {
		expect := map[string]bool{
			"stops.txt":   true,
			"missing.txt": false,
		}
		for k, v := range expect {
			found := false
			adapter.OpenFile(k, func(in io.Reader) { found = true })
			if found != v {
				t.Errorf("expected %t for %s", v, k)
			}
		}
	})
	t.Run("ReadRows", func(t *testing.T) {
		// TODO: more tests
		ent := tl.StopTime{}
		m := map[string]int{}
		total := 0
		adapter.ReadRows(ent.Filename(), func(row Row) {
			e := tl.StopTime{}
			loadRow(&e, row)
			m[e.StopID]++
			total++
		})
		expect := map[string]int{"EMSI": 2, "BULLFROG": 4, "STAGECOACH": 3}
		for k, v := range expect {
			if i := m[k]; v != i {
				t.Errorf("got %d for %s, expected %d", v, k, i)
			}
		}
		if total != 28 {
			t.Error("expected 28 rows, got ", total)
		}
	})
	t.Run("ReadRows-Malformed", func(t *testing.T) {
		errcount := 0
		expectrows := map[string]int{
			"valid":         6,
			"singlequoted":  2,
			"barequoted":    2,
			"bareendquote":  3,
			"openmultirow":  2,
			"morebadquotes": 3,
			"validend":      4,
		}
		foundrows := map[string]int{}
		adapter.ReadRows("malformed.txt", func(row Row) {
			// log.Debugf("%d %#v\n", len(row.Row), row.Row)
			if row.Err != nil {
				errcount++
			}
			foundrows[row.Row[0]] = len(row.Row)
		})
		for k, v := range expectrows {
			s, ok := foundrows[k]
			if !ok {
				t.Errorf("did not find expected row '%s'", k)
			}
			if s != v {
				t.Errorf("row '%s' got %d columns, expected %d", k, s, v)
			}
		}
		if errcount != 0 {
			t.Errorf("got %d errors, expected 3 parse errors from malformed test file", errcount)
		}
	})
	closeerr := adapter.Close()
	t.Run("Close", func(t *testing.T) {
		if closeerr != nil {
			t.Error(closeerr)
		}
	})
}
