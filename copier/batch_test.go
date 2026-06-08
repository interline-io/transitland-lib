package copier

import (
	"context"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/adapters/direct"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

// batchTripReader wraps a direct.Reader and adds TripsByID, so it satisfies the
// copier's tripBatchReader capability and drives the batched trip-loading path the
// same way a real tlcsv reader does. The embedded direct.Reader supplies the
// streaming StopTimesByTripID() the batched path reads from.
type batchTripReader struct {
	*direct.Reader
}

func (r *batchTripReader) TripsByID(ids ...string) chan gtfs.Trip {
	set := map[string]bool{}
	for _, id := range ids {
		set[id] = true
	}
	out := make(chan gtfs.Trip, 1000)
	go func() {
		for _, tr := range r.TripList {
			if len(set) == 0 || set[tr.TripID.Val] {
				out <- tr
			}
		}
		close(out)
	}()
	return out
}

// buildBatchTestReader returns a small feed exercising the cases that differ
// between the batched and cached trip-loading paths: trips whose stop_times span
// several chunks, a duplicate trip_id, and a trip with no stop_times at all.
func buildBatchTestReader() *direct.Reader {
	r := direct.NewReader()
	r.AgencyList = []gtfs.Agency{
		{AgencyID: tt.NewString("ag1"), AgencyName: tt.NewString("Ag"), AgencyTimezone: tt.NewTimezone("America/Los_Angeles"), AgencyURL: tt.NewUrl("http://example.com")},
	}
	r.RouteList = []gtfs.Route{
		{RouteID: tt.NewString("r1"), RouteType: tt.NewInt(3), AgencyID: tt.NewKey("ag1")},
	}
	r.CalendarList = []gtfs.Calendar{
		{ServiceID: tt.NewString("s1"), StartDate: tt.NewDate(time.Now()), EndDate: tt.NewDate(time.Now())},
	}
	r.StopList = []gtfs.Stop{
		{StopID: tt.NewString("a"), StopName: tt.NewString("A"), Geometry: tt.NewPoint(1, 1)},
		{StopID: tt.NewString("b"), StopName: tt.NewString("B"), Geometry: tt.NewPoint(2, 2)},
		{StopID: tt.NewString("c"), StopName: tt.NewString("C"), Geometry: tt.NewPoint(3, 3)},
	}
	addTrip := func(id string) {
		r.TripList = append(r.TripList, gtfs.Trip{TripID: tt.NewString(id), RouteID: tt.NewKey("r1"), ServiceID: tt.NewKey("s1")})
	}
	stops := []string{"a", "b", "c"}
	addStopTimes := func(tripID string, n int) {
		for i := 0; i < n; i++ {
			r.StopTimeList = append(r.StopTimeList, gtfs.StopTime{
				TripID:        tt.NewString(tripID),
				StopID:        tt.NewKey(stops[i%len(stops)]),
				StopSequence:  tt.NewInt(i + 1),
				ArrivalTime:   tt.NewSeconds(3600 + i*60),
				DepartureTime: tt.NewSeconds(3600 + i*60 + 30),
			})
		}
	}
	for _, tc := range []struct {
		id string
		n  int
	}{{"t1", 3}, {"t2", 2}, {"t3", 3}, {"t4", 2}, {"t5", 3}, {"tdup", 2}} {
		addTrip(tc.id)
		addStopTimes(tc.id, tc.n)
	}
	// Duplicate trip_id (second row, no stop_times of its own)
	addTrip("tdup")
	// Trip present in trips.txt but absent from stop_times
	addTrip("tnostops")
	return r
}

func batchCopyTrips(t *testing.T, reader adapters.Reader) ([]string, map[string]string) {
	t.Helper()
	writer := direct.NewWriter()
	opts := Options{
		// Mirror the rebuild-stats / fetch stats path, which is where the
		// all-trips cache OOMs.
		Quiet:                true,
		NoValidators:         true,
		AllowEntityErrors:    true,
		AllowReferenceErrors: true,
	}
	if _, err := CopyWithOptions(context.Background(), reader, writer, opts); err != nil {
		t.Fatal(err)
	}
	wr, err := writer.NewReader()
	if err != nil {
		t.Fatal(err)
	}
	var tripIDs []string
	for trip := range wr.Trips() {
		tripIDs = append(tripIDs, trip.TripID.Val)
	}
	sort.Strings(tripIDs)
	stopTimes := map[string]string{}
	for st := range wr.StopTimes() {
		stopTimes[st.TripID.Val+"/"+strconv.FormatInt(st.StopSequence.Val, 10)] = st.StopID.Val
	}
	return tripIDs, stopTimes
}

func TestCopier_TripBatchingEquivalence(t *testing.T) {
	// Cached path: a plain direct.Reader does not implement tripBatchReader.
	wantTrips, wantStopTimes := batchCopyTrips(t, buildBatchTestReader())

	// Sanity: the fixture writes the duplicate trip twice and the stop_time-less
	// trip once, so the comparison below actually covers those cases.
	assert.Equal(t, []string{"t1", "t2", "t3", "t4", "t5", "tdup", "tdup", "tnostops"}, wantTrips)

	// Batched path: the stop_time flush limit controls how many batches the trips
	// are reloaded in; any limit should produce identical output. A limit of 1
	// flushes after every trip group, exercising the many-batch path.
	defer func(old int) { tripBatchStopTimeLimit = old }(tripBatchStopTimeLimit)
	for _, limit := range []int{1, 2, 5, 1_000_000} {
		t.Run("limit="+strconv.Itoa(limit), func(t *testing.T) {
			tripBatchStopTimeLimit = limit
			reader := &batchTripReader{Reader: buildBatchTestReader()}
			gotTrips, gotStopTimes := batchCopyTrips(t, reader)
			assert.Equal(t, wantTrips, gotTrips, "trips written should match cached path")
			assert.Equal(t, wantStopTimes, gotStopTimes, "stop_times written should match cached path")
		})
	}
}
