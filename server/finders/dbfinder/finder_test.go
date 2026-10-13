package dbfinder

import (
	"context"
	"errors"
	"testing"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A keyed finder that fails returns an error for each key, which its dataloader
// hands to that key's caller.
func TestFinder_KeyedErrors(t *testing.T) {
	ctx := context.Background()
	// Every query fails on a finished transaction.
	tx, err := testutil.MustOpenTestDB(t).Beginx()
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())
	f := NewFinder(tx)

	// No query reaches the database, so the keys are arbitrary.
	ids := []int{1, 2, 3}
	fvKeys := []model.FVEntityID{{FeedVersionID: 1, EntityID: "a"}, {FeedVersionID: 1, EntityID: "b"}, {FeedVersionID: 2, EntityID: "a"}}
	places := []model.StopPlaceParam{{ID: 1}, {ID: 2}, {ID: 3}}
	errsOf := func(_ any, errs []error) []error { return errs }
	tcs := []struct {
		name string
		errs []error
	}{
		{"AgenciesByIDs", errsOf(f.AgenciesByIDs(ctx, ids))},
		{"AreasByIDs", errsOf(f.AreasByIDs(ctx, ids))},
		{"BookingRulesByIDs", errsOf(f.BookingRulesByIDs(ctx, ids))},
		{"CalendarsByIDs", errsOf(f.CalendarsByIDs(ctx, ids))},
		{"CensusDatasetLayersByDatasetIDs", errsOf(f.CensusDatasetLayersByDatasetIDs(ctx, ids))},
		{"CensusLayersByIDs", errsOf(f.CensusLayersByIDs(ctx, ids))},
		{"CensusSourceLayersBySourceIDs", errsOf(f.CensusSourceLayersBySourceIDs(ctx, ids))},
		{"CensusSourcesByIDs", errsOf(f.CensusSourcesByIDs(ctx, ids))},
		{"CensusTableByIDs", errsOf(f.CensusTableByIDs(ctx, ids))},
		{"FareAttributesByIDs", errsOf(f.FareAttributesByIDs(ctx, ids))},
		{"FareLegRulesByFeedVersionLegGroupIDs", errsOf(f.FareLegRulesByFeedVersionLegGroupIDs(ctx, fvKeys))},
		{"FareMediaByIDs", errsOf(f.FareMediaByIDs(ctx, ids))},
		{"FareProductsByFeedVersionFareProductIDs", errsOf(f.FareProductsByFeedVersionFareProductIDs(ctx, fvKeys))},
		{"FeedStatesByFeedIDs", errsOf(f.FeedStatesByFeedIDs(ctx, ids))},
		{"FeedVersionGeometryByIDs", errsOf(f.FeedVersionGeometryByIDs(ctx, ids))},
		{"FeedVersionGtfsImportByFeedVersionIDs", errsOf(f.FeedVersionGtfsImportByFeedVersionIDs(ctx, ids))},
		{"FeedVersionServiceWindowByFeedVersionIDs", errsOf(f.FeedVersionServiceWindowByFeedVersionIDs(ctx, ids))},
		{"FeedVersionsByIDs", errsOf(f.FeedVersionsByIDs(ctx, ids))},
		{"FeedsByIDs", errsOf(f.FeedsByIDs(ctx, ids))},
		{"LevelsByIDs", errsOf(f.LevelsByIDs(ctx, ids))},
		{"LocationGroupsByIDs", errsOf(f.LocationGroupsByIDs(ctx, ids))},
		{"LocationsByIDs", errsOf(f.LocationsByIDs(ctx, ids))},
		{"NetworksByIDs", errsOf(f.NetworksByIDs(ctx, ids))},
		{"OperatorsByAgencyIDs", errsOf(f.OperatorsByAgencyIDs(ctx, ids))},
		{"OperatorsByCOIFs", errsOf(f.OperatorsByCOIFs(ctx, ids))},
		{"PathwaysByIDs", errsOf(f.PathwaysByIDs(ctx, ids))},
		{"RiderCategoriesByFeedVersionRiderCategoryIDs", errsOf(f.RiderCategoriesByFeedVersionRiderCategoryIDs(ctx, fvKeys))},
		{"RouteAttributesByRouteIDs", errsOf(f.RouteAttributesByRouteIDs(ctx, ids))},
		{"RouteTypesByAgencyIDs", errsOf(f.RouteTypesByAgencyIDs(ctx, ids))},
		{"RoutesByFeedVersionRouteIDs", errsOf(f.RoutesByFeedVersionRouteIDs(ctx, fvKeys))},
		{"RoutesByIDs", errsOf(f.RoutesByIDs(ctx, ids))},
		{"SegmentsByIDs", errsOf(f.SegmentsByIDs(ctx, ids))},
		{"ShapesByIDs", errsOf(f.ShapesByIDs(ctx, ids))},
		{"StopExternalReferencesByStopIDs", errsOf(f.StopExternalReferencesByStopIDs(ctx, ids))},
		{"StopPlacesByStopID", errsOf(f.StopPlacesByStopID(ctx, places))},
		{"StopsByFeedVersionStopIDs", errsOf(f.StopsByFeedVersionStopIDs(ctx, fvKeys))},
		{"StopsByIDs", errsOf(f.StopsByIDs(ctx, ids))},
		{"TargetStopsByStopIDs", errsOf(f.TargetStopsByStopIDs(ctx, ids))},
		{"TimeframesByFeedVersionTimeframeGroupIDs", errsOf(f.TimeframesByFeedVersionTimeframeGroupIDs(ctx, fvKeys))},
		{"TripsByFeedVersionTripIDs", errsOf(f.TripsByFeedVersionTripIDs(ctx, fvKeys))},
		{"TripsByIDs", errsOf(f.TripsByIDs(ctx, ids))},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			require.Len(t, tc.errs, len(ids))
			for _, err := range tc.errs {
				assert.Error(t, err)
			}
		})
	}
}

func Test_logExtendErr(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	tcs := []struct {
		name   string
		ctx    context.Context
		expect string
	}{
		{"query failed", context.Background(), "database error"},
		{"request canceled", canceled, context.Canceled.Error()},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			errs := logExtendErr(tc.ctx, 3, errors.New("connection reset"))
			require.Len(t, errs, 3)
			for _, err := range errs {
				assert.EqualError(t, err, tc.expect)
			}
		})
	}
}

func TestFinder_FindFeedVersionServiceWindow(t *testing.T) {
	ctx := context.Background()
	db := testutil.MustOpenTestDB(t)
	dbf := NewFinder(dbutil.WithQueryLogger(db, false, 0))
	testFinder := dbf

	fvm := map[string]int{}
	fvs, err := testFinder.FindFeedVersions(ctx, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, fv := range fvs {
		fvm[fv.SHA1] = fv.ID
	}
	tcs := []struct {
		name  string
		fvid  int
		start string
		end   string
		best  string
	}{
		{
			"hart",
			fvm["c969427f56d3a645195dd8365cde6d7feae7e99b"],
			"2018-02-26", // calculated
			"2018-10-21",
			"2018-07-09",
		},
		{
			"bart",
			fvm["e535eb2b3b9ac3ef15d82c56575e914575e732e0"],
			"2018-05-26", // from feed info
			"2019-07-01", // from feed info
			"2018-06-04",
		},
		{
			"caltrain",
			fvm["d2813c293bcfd7a97dde599527ae6c62c98e66c6"],
			"2018-06-18", // calculated
			"2019-10-06",
			"2018-06-18",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			fvsw, err := testFinder.FindFeedVersionServiceWindow(ctx, tc.fvid)
			start, end, best := fvsw.StartDate, fvsw.EndDate, fvsw.FallbackWeek
			if err != nil {
				t.Fatal(err)
			}
			df := "2006-01-02"
			assert.Equal(t, tc.start, start.Format(df), "did not get expected window start")
			assert.Equal(t, tc.end, end.Format(df), "did not get expected window end")
			assert.Equal(t, tc.best, best.Format(df), "did not get expected best week in window")
			if end.Before(end) {
				t.Errorf("window end date %s before window start date %s", start.Format(df), end.Format(df))
			}
			if best.Before(start) {
				t.Errorf("best date %s before window start date %s", best.Format(df), start.Format(df))
			}
			if best.After(end) {
				t.Errorf("best date %s after window end date %s", best.Format(df), end.Format(df))
			}
		})
	}
}

func Test_alphanumeric(t *testing.T) {
	tcs := []struct {
		name   string
		value  string
		expect string
	}{
		{"ascii char", "a", "a"},
		{"ascii string", "abc", "abc"},
		{"ascii alphanumeric", "abc123", "abc123"},
		{"ascii space", "a b c", "a b c"},
		{"emdash to space", "a—b", "a b"},
		{"double space ok", "a  b", "a  b"},
		{"slash to space", "a/b", "a b"},
		{"single quote to space", "a'b", "a b"},
		{"double quote to space", "\"", " "},
		{": to space", "a:b", "a b"},
		{"* to space", "a*b", "a b"},
		{"& to space", "a&b", "a b"},
		{"| to space", "a|b", "a b"},
		{"hyphen to space", "Winston-Salem", "Winston Salem"},
		{"tab to space", "\t", " "},
		{"french", "Hôtel", "Hôtel"},
		{"combining mark kept", "Montre\u0301al", "Montre\u0301al"},
		{"devanagari vowel signs kept", "दिल्ली", "दिल्ली"},
		{"chinese", "火车", "火车"},
		{"chinese with ascii", "abc 火车 123", "abc 火车 123"},
		{"japanese", "列車", "列車"},
		{"russian", "тренироваться", "тренироваться"},
		{"hebrew", "רכבת", "רכבת"},
		{"arabic", "قطار", "قطار"},
		{"arabic with ascii", "test قطار", "test قطار"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ret := alphanumeric(tc.value)
			if ret != tc.expect {
				t.Errorf("got '%s', expect '%s'", ret, tc.expect)
			}
		})
	}
}

func Test_escapeWordsWithSuffix(t *testing.T) {
	tcs := []struct {
		name   string
		value  string
		expect []string
	}{
		{"words", "san fran", []string{"san:*", "fran:*"}},
		{"punctuation splits words", "Winston-Salem", []string{"Winston:*", "Salem:*"}},
		{"short pieces dropped", "coeur d'alene", []string{"coeur:*", "alene:*"}},
		{"nothing usable", "a b !!", nil},
		{"one accented letter is one character", "é", nil},
		{"two accented letters", "éé", []string{"éé:*"}},
		{"decomposed accent composed", "Montre\u0301al", []string{"Montr\u00e9al:*"}},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, escapeWordsWithSuffix(tc.value, ":*"))
		})
	}
}

func Test_az09(t *testing.T) {
	tcs := []struct {
		name   string
		value  string
		expect string
	}{
		{"plain", "hello", "hello"},
		{"underscore", "hello_world", "hello_world"},
		{"digits", "123", "123"},
		{"remove quotes", "a'b'\"c", "abc"},
		{"remove symbols", "a!b@c#d$e%f;g(h", "abcdefgh"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ret := az09(tc.value)
			if ret != tc.expect {
				t.Errorf("got '%s', expect '%s'", ret, tc.expect)
			}
		})
	}
}
