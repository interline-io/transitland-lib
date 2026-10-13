package rtfinder

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/internal/testreader"
	"github.com/interline-io/transitland-lib/rt"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/caches/kvcache"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/testdata"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

// countingDB counts the trip, feed version agency and route lookups run
// through it, and fails the trip and agency lookups while fail is set.
type countingDB struct {
	tldb.Ext
	tripQueries   atomic.Int64
	agencyQueries atomic.Int64
	routeQueries  atomic.Int64
	fail          atomic.Bool
}

func (db *countingDB) QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error) {
	counter := (*atomic.Int64)(nil)
	if strings.Contains(query, "gtfs_trips") {
		counter = &db.tripQueries
	} else if query == feedVersionAgencyIdsQuery {
		counter = &db.agencyQueries
	}
	if counter != nil {
		counter.Add(1)
		if db.fail.Load() {
			return nil, errors.New("test failure")
		}
	}
	return db.Ext.QueryxContext(ctx, query, args...)
}

func (db *countingDB) QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row {
	if query == feedVersionRouteQuery || query == routeQuery {
		db.routeQueries.Add(1)
	}
	return db.Ext.QueryRowxContext(ctx, query, args...)
}

// testBartFeedVersion returns a BART feed version holding the trips the RT
// fixtures name.
func testBartFeedVersion(t *testing.T, db tldb.Ext) int {
	fvid := 0
	if err := sqlx.Get(db, &fvid, `select max(feed_version_id) from gtfs_trips where trip_id = $1`, "1031527WKDY"); err != nil {
		t.Fatal(err)
	}
	return fvid
}

// testRoutes returns a feed version's routes by GTFS route_id.
func testRoutes(t *testing.T, db tldb.Ext, fvid int) map[string]*model.Route {
	var routes []*model.Route
	q := `select id, feed_version_id, route_id, agency_id, route_type from gtfs_routes where feed_version_id = $1`
	if err := sqlx.Select(db, &routes, q, fvid); err != nil {
		t.Fatal(err)
	}
	ret := map[string]*model.Route{}
	for _, r := range routes {
		ret[r.RouteID.Val] = r
	}
	return ret
}

// testStop returns a feed version's stop by GTFS stop_id.
func testStop(t *testing.T, db tldb.Ext, fvid int, stopId string) *model.Stop {
	stop := &model.Stop{}
	q := `select id, feed_version_id, stop_id from gtfs_stops where feed_version_id = $1 and stop_id = $2`
	if err := sqlx.Get(db, stop, q, fvid, stopId); err != nil {
		t.Fatal(err)
	}
	return stop
}

// testReadRT reads an RT fixture from testdata/server/rt.
func testReadRT(t *testing.T, fname string) *pb.FeedMessage {
	msg, err := rt.ReadFile(testdata.Path("server", "rt", fname))
	if err != nil {
		t.Fatal(err)
	}
	return msg
}

// testFinder returns a Finder over lookup holding msg as BART's ftype feed.
func testFinder(t *testing.T, lookup Lookup, ftype string, msg *pb.FeedMessage) *Finder {
	f := NewFinder(kvcache.NewMemoryStore(), lookup)
	t.Cleanup(func() { f.Close() })
	data, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.AddData(context.Background(), getTopicKey("BA", ftype), data); err != nil {
		t.Fatal(err)
	}
	return f
}

// A frequency-based trip runs from its first start to its last start plus one
// run, and no run starts at a frequency's end_time.
func TestPostgresLookup_GetTripSpan(t *testing.T) {
	if a, ok := testutil.CheckTestDB(); !ok {
		t.Skip(a)
	}
	tx, err := testutil.MustOpenTestDB(t).Beginx()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })
	fvid := testdb.CopyTestFeed(t, tx, testreader.NewFrequencyTestFeed())
	lookup := NewPostgresLookup(tx)
	for _, tc := range []struct {
		tripID string
		expect TripSpan
	}{
		{"regular", TripSpan{FirstDeparture: 5 * 3600, LastArrival: 5*3600 + 600}},
		{"offset", TripSpan{FirstDeparture: 7 * 3600, LastArrival: 7*3600 + 1800 + 600}},
		{"meet", TripSpan{FirstDeparture: 10 * 3600, LastArrival: 11*3600 + 1800 + 600}},
	} {
		t.Run(tc.tripID, func(t *testing.T) {
			tripID := 0
			if err := sqlx.Get(tx, &tripID, "select id from gtfs_trips where feed_version_id = $1 and trip_id = $2", fvid, tc.tripID); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.expect, lookup.GetTripSpan(context.Background(), fvid, tripID))
		})
	}
}

func TestPostgresLookup_GetGtfsStopID(t *testing.T) {
	if a, ok := testutil.CheckTestDB(); !ok {
		t.Skip(a)
	}
	db := testutil.MustOpenTestDB(t)
	stop := testStop(t, db, testBartFeedVersion(t, db), "FTVL")
	lookup := NewPostgresLookup(db)
	ctx := context.Background()
	sid, ok := lookup.GetGtfsStopID(ctx, stop.ID)
	assert.True(t, ok)
	assert.Equal(t, "FTVL", sid)
	// A failed lookup is not cached, where its empty stop_id would match a stop
	// time update naming no stop.
	for range 2 {
		_, ok := lookup.GetGtfsStopID(ctx, -1)
		assert.False(t, ok)
	}
}

func TestPostgresLookup_GetGtfsAgencyID(t *testing.T) {
	if a, ok := testutil.CheckTestDB(); !ok {
		t.Skip(a)
	}
	db := testutil.MustOpenTestDB(t)
	id := 0
	q := `select id from gtfs_agencies where feed_version_id = $1 and agency_id = $2`
	if err := sqlx.Get(db, &id, q, testBartFeedVersion(t, db), "BART"); err != nil {
		t.Fatal(err)
	}
	lookup := NewPostgresLookup(db)
	ctx := context.Background()
	aid, ok := lookup.GetGtfsAgencyID(ctx, id)
	assert.True(t, ok)
	assert.Equal(t, "BART", aid)
	// A failed lookup is not cached, where its empty agency_id would match any
	// agency.
	for range 2 {
		_, ok := lookup.GetGtfsAgencyID(ctx, -1)
		assert.False(t, ok)
	}
}

func TestPostgresLookup_GetFeedVersionRoute(t *testing.T) {
	if a, ok := testutil.CheckTestDB(); !ok {
		t.Skip(a)
	}
	raw := testutil.MustOpenTestDB(t)
	db := &countingDB{Ext: raw}
	fvid := testBartFeedVersion(t, raw)
	lookup := NewPostgresLookup(db)
	ctx := context.Background()
	r, ok := lookup.GetFeedVersionRoute(ctx, fvid, "05")
	if assert.True(t, ok) {
		assert.NotZero(t, r.ID)
		assert.Equal(t, "05", r.RouteID)
		assert.Equal(t, 1, r.RouteType)
		assert.Equal(t, "BART", r.AgencyID)
	}
	byId, ok := lookup.GetRoute(ctx, r.ID)
	assert.True(t, ok)
	assert.Equal(t, r, byId)
	// A route that isn't there is looked up once, and remembered as missing.
	for range 2 {
		_, ok := lookup.GetFeedVersionRoute(ctx, fvid, "99")
		assert.False(t, ok)
		_, ok = lookup.GetRoute(ctx, -1)
		assert.False(t, ok)
	}
	assert.EqualValues(t, 4, db.routeQueries.Load())
}

func TestPostgresLookup_GetFeedVersionAgencyIDs(t *testing.T) {
	if a, ok := testutil.CheckTestDB(); !ok {
		t.Skip(a)
	}
	db := testutil.MustOpenTestDB(t)
	fvid := testBartFeedVersion(t, db)
	lookup := NewPostgresLookup(db)
	ctx := context.Background()
	agencyIds, ok := lookup.GetFeedVersionAgencyIDs(ctx, fvid)
	assert.True(t, ok)
	assert.Equal(t, []string{"BART"}, agencyIds)
	routeTypes, ok := lookup.GetFeedVersionRouteTypes(ctx, fvid)
	assert.True(t, ok)
	assert.Equal(t, []int{1}, routeTypes)
}
