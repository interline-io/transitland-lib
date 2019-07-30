package gtcsv

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/interline-io/gotransit/internal/testutil"
)

func NewExampleReader() (*testutil.ReaderTester, *Reader) {
	fe := testutil.ExampleFeed
	reader, err := NewReader(fe.URL)
	if err != nil {
		panic(err)
	}
	return &fe, reader
}

func TestReader(t *testing.T) {
	t.Run("Dir", func(t *testing.T) {
		fe, r := NewExampleReader()
		if err := r.Open(); err != nil {
			t.Error(err)
		}
		defer r.Close()
		fe.Test(t, r)
	})
	t.Run("Zip", func(t *testing.T) {
		fe, _ := NewExampleReader()
		reader, err := NewReader("../testdata/example.zip")
		if err != nil {
			t.Error(err)
		}
		if err := reader.Open(); err != nil {
			t.Error(err)
		}
		defer reader.Close()
		fe.Test(t, reader)
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
		fe, _ := NewExampleReader()
		reader, err := NewReader(ts.URL)
		if err != nil {
			t.Error(err)
		}
		if err := reader.Open(); err != nil {
			t.Error(err)
		}
		defer reader.Close()
		fe.Test(t, reader)
	})
}
