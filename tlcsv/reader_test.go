package tlcsv

import (
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
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
	var order []string        // trip_ids in yield order
	stopTimeCount := 0
	for tst := range reader.TripsWithStopTimes() {
		if !tst.Valid {
			t.Errorf("unexpected invalid entry with %d stop_times", len(tst.StopTimes))
			continue
		}
		trips[tst.Trip.TripID.Val]++
		order = append(order, tst.Trip.TripID.Val)
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

	// Trips are yielded in stop_times.txt first-appearance order, even though
	// chunkSize=5 splits them across several chunks. Order-sensitive validators
	// (block-overlap) rely on this being deterministic.
	want := []string{"STBA", "CITY1", "CITY2", "AB1", "AB2", "BFC1", "BFC2", "AAMV1", "AAMV2", "AAMV3", "AAMV4"}
	if !slices.Equal(order, want) {
		t.Errorf("trip order = %v, want %v", order, want)
	}

	// With ids, only those trips are yielded, still in trips.txt file order.
	var filtered []string
	for tst := range reader.TripsWithStopTimes("BFC2", "AB1") {
		if !tst.Valid || len(tst.StopTimes) == 0 {
			t.Errorf("filtered: unexpected entry valid=%v stop_times=%d", tst.Valid, len(tst.StopTimes))
		}
		filtered = append(filtered, tst.Trip.TripID.Val)
	}
	if wantFiltered := []string{"AB1", "BFC2"}; !slices.Equal(filtered, wantFiltered) {
		t.Errorf("filtered trips = %v, want %v", filtered, wantFiltered)
	}
}

// TestReader_TripsWithStopTimes_Grouping checks that the grouped single-pass fast
// path and the ungrouped chunked path produce identical output for the same logical
// feed, both in stop_times first-appearance order.
func TestReader_TripsWithStopTimes_Grouping(t *testing.T) {
	old := chunkSize
	chunkSize = 3 // force several chunks for 3 trips x 2 stop_times
	defer func() { chunkSize = old }()

	writeFeed := func(t *testing.T, stopTimes [][]string) string {
		t.Helper()
		dir := t.TempDir()
		w := NewDirAdapter(dir)
		if err := w.WriteRows("trips.txt", [][]string{
			{"route_id", "service_id", "trip_id"},
			{"r", "s", "t1"},
			{"r", "s", "t2"},
			{"r", "s", "t3"},
		}); err != nil {
			t.Fatal(err)
		}
		if err := w.WriteRows("stop_times.txt", append(
			[][]string{{"trip_id", "stop_id", "stop_sequence", "arrival_time", "departure_time"}},
			stopTimes...,
		)); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	st := func(trip, seq string) []string { return []string{trip, "a", seq, "08:00:00", "08:00:00"} }

	read := func(t *testing.T, dir string) ([]string, map[string][]int64) {
		t.Helper()
		reader, err := NewReader(dir)
		if err != nil {
			t.Fatal(err)
		}
		if err := reader.Open(); err != nil {
			t.Fatal(err)
		}
		defer reader.Close()
		var order []string
		byTrip := map[string][]int64{}
		for tst := range reader.TripsWithStopTimes() {
			if !tst.Valid {
				t.Fatalf("unexpected invalid entry with %d stop_times", len(tst.StopTimes))
			}
			order = append(order, tst.Trip.TripID.Val)
			for _, s := range tst.StopTimes {
				byTrip[tst.Trip.TripID.Val] = append(byTrip[tst.Trip.TripID.Val], s.StopSequence.Val)
			}
		}
		return order, byTrip
	}

	// Same logical feed, stop_times physically grouped vs interleaved by trip_id.
	grouped := writeFeed(t, [][]string{st("t1", "1"), st("t1", "2"), st("t2", "1"), st("t2", "2"), st("t3", "1"), st("t3", "2")})
	interleaved := writeFeed(t, [][]string{st("t1", "1"), st("t2", "1"), st("t3", "1"), st("t1", "2"), st("t2", "2"), st("t3", "2")})

	gOrder, gByTrip := read(t, grouped)
	iOrder, iByTrip := read(t, interleaved)

	want := []string{"t1", "t2", "t3"}
	if !slices.Equal(gOrder, want) {
		t.Errorf("grouped order = %v, want %v", gOrder, want)
	}
	if !slices.Equal(iOrder, want) {
		t.Errorf("interleaved order = %v, want %v", iOrder, want)
	}
	// The fast path and chunked path must agree on the joined data.
	if len(gByTrip) != len(iByTrip) {
		t.Fatalf("trip count differs: grouped %d, interleaved %d", len(gByTrip), len(iByTrip))
	}
	for trip, gSeqs := range gByTrip {
		if !slices.Equal(gSeqs, iByTrip[trip]) {
			t.Errorf("trip %s: grouped %v != interleaved %v", trip, gSeqs, iByTrip[trip])
		}
		if !slices.Equal(gSeqs, []int64{1, 2}) {
			t.Errorf("trip %s seqs = %v, want [1 2]", trip, gSeqs)
		}
	}
}

func TestReader_ShapesByShapeID_Order(t *testing.T) {
	// Grouped file (example feed): shapes emit in first-appearance order via the
	// single-pass fast path.
	t.Run("grouped", func(t *testing.T) {
		reader, err := NewReader(testreader.ExampleDir.URL)
		if err != nil {
			t.Fatal(err)
		}
		if err := reader.Open(); err != nil {
			t.Fatal(err)
		}
		defer reader.Close()
		var order []string
		for grp := range reader.ShapesByShapeID() {
			order = append(order, grp[0].ShapeID.Val)
		}
		if want := []string{"ok", "a", "c"}; !slices.Equal(order, want) {
			t.Errorf("grouped shape order = %v, want %v", order, want)
		}
	})

	// Interleaved file forces the non-grouped chunked path. With chunkSize=3 the two
	// shapes land in separate chunks but must still emit in first-appearance order,
	// each shape's points sorted by sequence (written here out of order on purpose).
	t.Run("interleaved", func(t *testing.T) {
		dir := t.TempDir()
		w := NewDirAdapter(dir)
		if err := w.WriteRows("shapes.txt", [][]string{
			{"shape_id", "shape_pt_lat", "shape_pt_lon", "shape_pt_sequence"},
			{"s1", "1", "1", "2"},
			{"s2", "2", "2", "1"},
			{"s1", "1", "1", "1"},
			{"s2", "2", "2", "2"},
			{"s1", "1", "1", "3"},
			{"s2", "2", "2", "3"},
		}); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}

		old := chunkSize
		chunkSize = 3
		defer func() { chunkSize = old }()

		reader, err := NewReader(dir)
		if err != nil {
			t.Fatal(err)
		}
		if err := reader.Open(); err != nil {
			t.Fatal(err)
		}
		defer reader.Close()

		var order []string
		for grp := range reader.ShapesByShapeID() {
			order = append(order, grp[0].ShapeID.Val)
			var last int64 = -1
			for _, s := range grp {
				if s.ShapePtSequence.Val < last {
					t.Errorf("shape %q points not sorted by sequence", grp[0].ShapeID.Val)
				}
				last = s.ShapePtSequence.Val
			}
		}
		if want := []string{"s1", "s2"}; !slices.Equal(order, want) {
			t.Errorf("interleaved shape order = %v, want %v", order, want)
		}
	})
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
