package rtfinder

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

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

// testFinder returns a Finder over db holding msg as BART's ftype feed.
func testFinder(t *testing.T, db tldb.Ext, ftype string, msg *pb.FeedMessage) *Finder {
	f := NewFinder(kvcache.NewMemoryStore(), db)
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

func TestLookupCache_GetTripRouteIDs(t *testing.T) {
	if a, ok := testutil.CheckTestDB(); !ok {
		t.Skip(a)
	}
	raw := testutil.MustOpenTestDB(t)
	fvid := testBartFeedVersion(t, raw)
	db := &countingDB{Ext: raw}
	lc := newLookupCache(db)
	ctx := context.Background()
	newSource := func(tripIds ...string) *Source {
		msg := &pb.FeedMessage{}
		for _, tid := range tripIds {
			msg.Entity = append(msg.Entity, &pb.FeedEntity{Alert: &pb.Alert{InformedEntity: []*pb.EntitySelector{{Trip: testTrip(tid, "")}}}})
		}
		src, err := NewSource("BA")
		if err != nil {
			t.Fatal(err)
		}
		if err := src.processMessage(ctx, msg); err != nil {
			t.Fatal(err)
		}
		return src
	}
	expect := map[string]string{"1031527WKDY": "05"}

	// Concurrent callers share one query, which resolves only the feed
	// version's own trips.
	src := newSource("1031527WKDY", "no-such-trip")
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			assert.Equal(t, expect, lc.GetTripRouteIDs(ctx, src, fvid))
		})
	}
	wg.Wait()
	assert.EqualValues(t, 1, db.tripQueries.Load(), "concurrent callers")

	// Another feed version resolves the same message on its own.
	assert.Empty(t, lc.GetTripRouteIDs(ctx, src, -1))
	assert.EqualValues(t, 2, db.tripQueries.Load(), "other feed version")

	// A message naming every trip's route needs no query.
	assert.Nil(t, lc.GetTripRouteIDs(ctx, newSource(), fvid))
	assert.EqualValues(t, 2, db.tripQueries.Load(), "nothing unrouted")

	// A caller that has gone away is served what is held, but starts no lookup.
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	assert.Equal(t, expect, lc.GetTripRouteIDs(canceled, src, fvid))
	assert.Nil(t, lc.GetTripRouteIDs(canceled, newSource("1031527WKDY"), fvid))
	assert.EqualValues(t, 2, db.tripQueries.Load(), "canceled")

	// A failed lookup is not cached.
	src = newSource("1031527WKDY")
	db.fail.Store(true)
	assert.Nil(t, lc.GetTripRouteIDs(ctx, src, fvid))
	db.fail.Store(false)
	assert.Equal(t, expect, lc.GetTripRouteIDs(ctx, src, fvid))
	assert.EqualValues(t, 4, db.tripQueries.Load(), "retry after failure")
}

func TestLookupCache_GetGtfsAgencyID(t *testing.T) {
	if a, ok := testutil.CheckTestDB(); !ok {
		t.Skip(a)
	}
	db := testutil.MustOpenTestDB(t)
	id := 0
	q := `select id from gtfs_agencies where feed_version_id = $1 and agency_id = $2`
	if err := sqlx.Get(db, &id, q, testBartFeedVersion(t, db), "BART"); err != nil {
		t.Fatal(err)
	}
	lc := newLookupCache(db)
	ctx := context.Background()
	aid, ok := lc.GetGtfsAgencyID(ctx, id)
	assert.True(t, ok)
	assert.Equal(t, "BART", aid)
	// A failed lookup is not cached, where its empty agency_id would match any
	// agency.
	for range 2 {
		_, ok := lc.GetGtfsAgencyID(ctx, -1)
		assert.False(t, ok)
	}
}

func TestLookupCache_GetFeedVersionRoute(t *testing.T) {
	if a, ok := testutil.CheckTestDB(); !ok {
		t.Skip(a)
	}
	raw := testutil.MustOpenTestDB(t)
	db := &countingDB{Ext: raw}
	fvid := testBartFeedVersion(t, raw)
	lc := newLookupCache(db)
	ctx := context.Background()
	r, ok := lc.GetFeedVersionRoute(ctx, fvid, "05")
	if assert.True(t, ok) {
		assert.NotZero(t, r.ID)
		assert.Equal(t, "05", r.RouteID)
		assert.Equal(t, 1, r.RouteType)
		assert.Equal(t, "BART", r.AgencyID)
	}
	byId, ok := lc.GetRoute(ctx, r.ID)
	assert.True(t, ok)
	assert.Equal(t, r, byId)
	// A route that isn't there is looked up once, and remembered as missing.
	for range 2 {
		_, ok := lc.GetFeedVersionRoute(ctx, fvid, "99")
		assert.False(t, ok)
		_, ok = lc.GetRoute(ctx, -1)
		assert.False(t, ok)
	}
	assert.EqualValues(t, 4, db.routeQueries.Load())
}

func TestLookupCache_GetFeedVersionAgencyIDs(t *testing.T) {
	if a, ok := testutil.CheckTestDB(); !ok {
		t.Skip(a)
	}
	db := testutil.MustOpenTestDB(t)
	fvid := testBartFeedVersion(t, db)
	lc := newLookupCache(db)
	ctx := context.Background()
	agencyIds, ok := lc.GetFeedVersionAgencyIDs(ctx, fvid)
	assert.True(t, ok)
	assert.Equal(t, []string{"BART"}, agencyIds)
	routeTypes, ok := lc.GetFeedVersionRouteTypes(ctx, fvid)
	assert.True(t, ok)
	assert.Equal(t, []int{1}, routeTypes)
}
