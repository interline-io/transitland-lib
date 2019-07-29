package gtcsv

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/interline-io/gotransit/internal/testutil"
)

func NewTestFeed(name string) (*testutil.ExpectEntities, *Reader) {
	fe, ok := testutil.TestFeed(name)
	if !ok {
		panic("no such example feed")
	}
	reader, err := NewReader(fe.URL)
	if err != nil {
		panic(err)
	}
	return &fe, reader
}

func TestReader(t *testing.T) {
	t.Run("Dir", func(t *testing.T) {
		fe, r := NewTestFeed("example")
		if err := r.Open(); err != nil {
			t.Error(err)
		}
		defer r.Close()
		testutil.CheckExpectEntities(t, *fe, r)
	})
	t.Run("Zip", func(t *testing.T) {
		fe, _ := NewTestFeed("example")
		reader, err := NewReader("../testdata/example.zip")
		if err != nil {
			t.Error(err)
		}
		if err := reader.Open(); err != nil {
			t.Error(err)
		}
		defer reader.Close()
		testutil.CheckExpectEntities(t, *fe, reader)
	})
	t.Run("URL", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			buf, err := ioutil.ReadFile("../testdata/example.zip")
			if err != nil {
				t.Error(err)
			}
			w.Write(buf)
		}))
		defer ts.Close()
		//
		fe, _ := NewTestFeed("example")
		reader, err := NewReader(ts.URL)
		if err != nil {
			t.Error(err)
		}
		if err := reader.Open(); err != nil {
			t.Error(err)
		}
		defer reader.Close()
		testutil.CheckExpectEntities(t, *fe, reader)
	})
}
