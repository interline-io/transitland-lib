package gtcsv

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/internal/testutil"
)

func TestReader(t *testing.T) {
	// Start local HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile("../testdata/example.zip")
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))
	defer ts.Close()
	//
	tsa := getTestAdapters()
	tsa["URL"] = func() Adapter { return &URLAdapter{url: ts.URL} }
	for k, v := range tsa {
		t.Run(k, func(t *testing.T) {
			testutil.TestReader(t, testutil.ExampleFeed, func() gotransit.Reader {
				return &Reader{Adapter: v()}
			})
		})
	}
}

func TestEntityErrors(t *testing.T) {
	reader, err := NewReader("../testdata/bad-entities")
	if err != nil {
		t.Error(err)
	}
	if err := reader.Open(); err != nil {
		t.Error(err)
	}
	testutil.TestEntityErrors(t, reader)
	if err := reader.Close(); err != nil {
		t.Error(err)
	}
}
