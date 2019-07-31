package gtcsv

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/interline-io/gotransit"
)

func compareMap(t *testing.T, result map[string]int, expect map[string]int) {
	for k, v := range expect {
		if i := result[k]; v != i {
			t.Error("expeced", k, "=", i)
		}
	}
}

// Test adapters
func Test_ZipAdapter_Exists(t *testing.T) {
	expect := map[string]bool{
		"../testdata/example.zip": true,
		"../testdata/missing.zip": false,
		"../test":                 false, // dir
		"../testdata/missing":     false, // dir
	}
	for k, v := range expect {
		r := ZipAdapter{path: k}
		if r.Exists() != v {
			t.Error("expected", v, "for", k)
		}
	}
}

func Test_ZipAdapter_OpenFile(t *testing.T) {
	r := ZipAdapter{path: "../testdata/example.zip"}
	expect := map[string]bool{
		"stops.txt":   true,
		"missing.txt": false,
	}
	for k, v := range expect {
		found := false
		r.OpenFile(k, func(in io.Reader) { found = true })
		if found != v {
			t.Error("expected", v, "for", k)
		}
	}
}

func Test_DirAdapter_Exists(t *testing.T) {
	expect := map[string]bool{
		"../testdata/example.zip": false,
		"../testdata/missing.zip": false,
		"../testdata/example":     true,  // dir
		"../testdata/missing":     false, // dir
	}
	for k, v := range expect {
		r := DirAdapter{path: k}
		if r.Exists() != v {
			t.Error("expected", v, "for", k)
		}
	}
}

func Test_DirAdapter_OpenFile(t *testing.T) {
	r := DirAdapter{path: "../testdata/example"}
	expect := map[string]bool{
		"stops.txt":   true,
		"missing.txt": false,
	}
	for k, v := range expect {
		found := false
		r.OpenFile(k, func(in io.Reader) { found = true })
		if found != v {
			t.Error("expected", v, "for", k)
		}
	}
}

func Test_URLAdapter_Download(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile("../testdata/example.zip")
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))
	defer ts.Close()
	adapter := URLAdapter{url: ts.URL}
	if err := adapter.Open(); err != nil {
		t.Error(err)
	}
	p := adapter.Path()
	if _, err := os.Stat(p); err != nil {
		t.Error("did not download file")
	}
	if err := adapter.Close(); err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(p); err == nil {
		t.Error("did not remove temp file")
	}
}

func TestDirAdapter_ReadRows(t *testing.T) {
	// TODO: more tests
	adapter := DirAdapter{path: "../testdata/example"}
	ent := gotransit.StopTime{}
	m := map[string]int{}
	total := 0
	adapter.ReadRows(ent.Filename(), func(row Row) {
		e := gotransit.StopTime{}
		loadRow(&e, row)
		m[e.StopID]++
		total++
	})
	expect := map[string]int{"EMSI": 2, "BULLFROG": 4, "STAGECOACH": 3}
	compareMap(t, m, expect)
	if total != 28 {
		t.Error("expected 28 rows, got ", total)
	}
}

func TestDirAdapter_ReadRows_errors(t *testing.T) {
	adapter := DirAdapter{path: "../testdata/example"}
	count := 0
	errcount := 0
	adapter.ReadRows("malformed.txt", func(row Row) {
		if row.Err != nil {
			errcount++
		}
		count++
	})
	if count < 4 {
		t.Error("expected at least 4 rows in malformed csv test file")
	}
	if errcount != 3 {
		t.Error("expected 3 parse errors from malformed csv test file")
	}
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
