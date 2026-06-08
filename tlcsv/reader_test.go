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

func TestReader_TripsWithStopTimes(t *testing.T) {
	reader, err := NewReader(testreader.ExampleDir.URL)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Open(); err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	// Force small chunks so the example feed's trips span several chunks.
	old := chunkSize
	chunkSize = 5
	defer func() { chunkSize = old }()

	trips := map[string]int{} // trip_id -> times yielded
	stopTimeCount := 0
	for tst := range reader.TripsWithStopTimes() {
		if !tst.Valid {
			t.Errorf("unexpected invalid entry with %d stop_times", len(tst.StopTimes))
			continue
		}
		trips[tst.Trip.TripID.Val]++
		stopTimeCount += len(tst.StopTimes)
		// Stop_times belong to this trip and are sorted by stop_sequence.
		var last int64 = -1
		for _, st := range tst.StopTimes {
			if st.TripID.Val != tst.Trip.TripID.Val {
				t.Errorf("stop_time trip_id %q under trip %q", st.TripID.Val, tst.Trip.TripID.Val)
			}
			if st.StopSequence.Val < last {
				t.Errorf("stop_times not sorted for trip %q", tst.Trip.TripID.Val)
			}
			last = st.StopSequence.Val
		}
	}

	// All 11 example-feed trips have stop_times; each is yielded exactly once.
	if len(trips) != 11 {
		t.Errorf("yielded %d distinct trips, want 11", len(trips))
	}
	for id, n := range trips {
		if n != 1 {
			t.Errorf("trip %q yielded %d times, want 1", id, n)
		}
	}
	if stopTimeCount != 28 {
		t.Errorf("yielded %d stop_times, want 28", stopTimeCount)
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
