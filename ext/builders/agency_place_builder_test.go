package builders

import (
	"context"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/adapters/direct"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/internal/testreader"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/stretchr/testify/assert"
)

// runAgencyPlace copies a feed through an AgencyPlaceBuilder and returns it for
// white-box inspection of the accumulated per-agency geohash tally. The Copy step
// (which needs a postgres writer) is intentionally not exercised here; the refactor
// only affects how AfterWrite tallies stop visits.
func runAgencyPlace(t *testing.T, url string, dedup bool) *AgencyPlaceBuilder {
	t.Helper()
	reader, err := tlcsv.NewReader(url)
	if err != nil {
		t.Fatal(err)
	}
	opts := copier.Options{DeduplicateJourneyPatterns: dedup}
	e := NewAgencyPlaceBuilder(nil)
	opts.AddExtension(e)
	if _, err := copier.CopyWithOptions(context.Background(), reader, direct.NewWriter(), opts); err != nil {
		t.Fatal(err)
	}
	return e
}

func agencyVisits(b *AgencyPlaceBuilder, aid string) (geohashes int, total int) {
	gh := b.agencyStops[aid]
	for _, c := range gh {
		total += c
	}
	return len(gh), total
}

func TestAgencyPlaceBuilder(t *testing.T) {
	// The builder tallies one visit per valid stop_time across the agency's trips,
	// reading each trip's own stop_times in its AfterWrite and resolving stop ids
	// through the EntityMap. BART is a single-agency feed, so every stop_time counts
	// toward BART.
	t.Run("tallies stop visits per agency", func(t *testing.T) {
		b := runAgencyPlace(t, testreader.ExampleFeedBART.URL, false)
		geohashes, total := agencyVisits(b, "BART")
		assert.Equal(t, 48, geohashes, "distinct stop geohashes for BART")
		assert.Equal(t, 33167, total, "total stop_time visits for BART")
	})

	// Because the tally is driven by each trip's own stop_times in the Trip AfterWrite
	// (not by individually written StopTime entities), journey-pattern deduplication —
	// which drops duplicate trips' written stop_times — does not change the agency
	// tally. Deduplicated trips are still counted, weighting by service frequency.
	t.Run("deduplication does not change the tally", func(t *testing.T) {
		_, totalOff := agencyVisits(runAgencyPlace(t, testreader.ExampleFeedBART.URL, false), "BART")
		_, totalOn := agencyVisits(runAgencyPlace(t, testreader.ExampleFeedBART.URL, true), "BART")
		assert.Equal(t, totalOff, totalOn, "dedup on/off must tally the same")
	})
}

// The Postgres lookup reads the database's Natural Earth tables.
func TestAgencyPlaceBuilder_PostgresPlaces(t *testing.T) {
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	if dburl == "" {
		t.Skip("TL_TEST_DATABASE_URL is not set")
	}
	testdb.TempPostgresRollback(t, dburl, func(atx tldb.Adapter) {
		ctx := context.Background()
		p := postgresPlaces{db: atx}
		sf := tlxy.Point{Lon: -122.4194, Lat: 37.7749}
		place, ok, err := p.NearestPlace(ctx, sf, placeRadius)
		if err != nil {
			t.Fatal(err)
		}
		assert.True(t, ok)
		assert.Equal(t, "San Francisco", place.Name.Val)
		assert.Equal(t, "California", place.Adm1name.Val)
		admin, ok, err := p.Admin(ctx, sf)
		if err != nil {
			t.Fatal(err)
		}
		assert.True(t, ok)
		assert.Equal(t, "California", admin.Adm1name.Val)
		assert.Equal(t, "United States of America", admin.Adm0name.Val)
		// Mid-Pacific, far from any populated place.
		_, ok, err = p.NearestPlace(ctx, tlxy.Point{Lon: -140, Lat: 30}, placeRadius)
		assert.NoError(t, err)
		assert.False(t, ok)
	})
}

// With a place lookup, the builder writes places through any writer. Every BART
// stop lies in the one admin area here, which then holds all of BART's visits.
func TestAgencyPlaceBuilder_PlaceLookup(t *testing.T) {
	usa := "United States of America"
	ix, err := NewPlaceIndex(
		[]AdminArea{{Name: "California", Admin: usa, Geometry: rect(-123, 37, -121, 38.5)}},
		[]PopulatedPlace{
			{Name: "San Francisco", Adm1Name: "California", Adm0Name: usa, Point: tlxy.Point{Lon: -122.4194, Lat: 37.7749}},
			{Name: "Oakland", Adm1Name: "California", Adm0Name: usa, Point: tlxy.Point{Lon: -122.2712, Lat: 37.8044}},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := tlcsv.NewReader(testreader.ExampleFeedBART.URL)
	if err != nil {
		t.Fatal(err)
	}
	opts := copier.Options{}
	opts.AddExtension(NewAgencyPlaceBuilder(ix))
	w := direct.NewWriter()
	if _, err := copier.CopyWithOptions(context.Background(), reader, w, opts); err != nil {
		t.Fatal(err)
	}
	got := map[string]*AgencyPlace{}
	for _, ent := range w.Reader.OtherList {
		if ap, ok := ent.(*AgencyPlace); ok {
			got[ap.Name.Val] = ap
		}
	}
	area, ok := got[""]
	if !ok {
		t.Fatal("no place for BART's admin area")
	}
	assert.Equal(t, "BART", area.AgencyID)
	assert.Equal(t, "California", area.Adm1name.Val)
	assert.Equal(t, 1.0, area.Rank)
	assert.Equal(t, 33167, area.Count)
	for _, name := range []string{"San Francisco", "Oakland"} {
		if p, ok := got[name]; assert.True(t, ok, name) {
			assert.Equal(t, usa, p.Adm0name.Val, name)
			assert.Greater(t, p.Rank, 0.05, name)
		}
	}
}
