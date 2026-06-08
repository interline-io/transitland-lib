package tlcsv

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/internal/testreader"
	"github.com/interline-io/transitland-lib/request"
)

func TestReader_TripsByID(t *testing.T) {
	reader, err := NewReader(testreader.ExampleDir.URL)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Open(); err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	// No filter yields all trips, matching Trips().
	all := 0
	for range reader.TripsByID() {
		all++
	}
	if all != 11 {
		t.Errorf("TripsByID() yielded %d trips, want 11", all)
	}

	// Filtered yields only the requested trips.
	got := map[string]bool{}
	for trip := range reader.TripsByID("AB1", "STBA") {
		got[trip.TripID.Val] = true
	}
	if len(got) != 2 || !got["AB1"] || !got["STBA"] {
		t.Errorf("TripsByID(AB1,STBA) = %v, want {AB1, STBA}", got)
	}

	// Unknown id yields nothing.
	none := 0
	for range reader.TripsByID("does-not-exist") {
		none++
	}
	if none != 0 {
		t.Errorf("TripsByID(unknown) yielded %d trips, want 0", none)
	}
}

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
	tsa["URL"] = func() Adapter {
		return &URLAdapter{url: ts.URL, reqOpts: []request.RequestOption{request.WithAllowHTTPUnfiltered}}
	}
	for k, v := range tsa {
		t.Run(k, func(t *testing.T) {
			testreader.TestReader(t, testreader.ExampleDir, func() adapters.Reader {
				return &Reader{Adapter: v()}
			})
		})
	}
}
