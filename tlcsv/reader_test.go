package tlcsv

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/internal/testreader"
)

func TestReader(t *testing.T) {
	// Start local HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := os.ReadFile(testreader.ExampleZip.URL)
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
			testreader.TestReader(t, testreader.ExampleDir, func() adapters.Reader {
				return &Reader{Adapter: v()}
			})
		})
	}
}
