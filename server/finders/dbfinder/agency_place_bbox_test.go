package dbfinder

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/testutil"
	sq "github.com/irees/squirrel"
	"github.com/stretchr/testify/assert"
)

// Exercises placeBboxSQL against the Natural Earth tables directly, which needs
// no imported feed: the aggregation levels reach this expression through a join
// on tl_agency_places, but the expression itself only ever sees the geometry the
// lateral hands it. Places crossing the antimeridian are the reason it exists and
// no fixture feed has operators near the line.
func TestPlaceBboxSQL(t *testing.T) {
	dbx := testutil.MustOpenTestDB(t)
	tcs := []struct {
		name    string
		city    bool
		place   string
		minLon  float64
		maxLon  float64
		comment string
	}{
		{
			name: "region crossing the antimeridian", place: "Alaska",
			minLon: 172.476, maxLon: 230.011,
			comment: "Attu east to the panhandle: 57 degrees, not the 359 the normal frame reports",
		},
		{
			name: "region away from the line keeps its coordinates", place: "California",
			minLon: -124.409, maxLon: -114.119,
			comment: "both frames measure the same width, so the tie keeps this within [-180,180]",
		},
		{
			name: "city buffer crossing the antimeridian", city: true, place: "Beringovskiy",
			minLon: 178.515, maxLon: 180.097,
			comment: "40km around a Chukotka town reaches past the line",
		},
		{
			name: "city buffer near the line but not crossing", city: true, place: "Funafuti",
			minLon: 178.853, maxLon: 179.580,
			comment: "close enough to matter, far enough that the normal frame is still narrower",
		},
		{
			name: "city away from the line keeps its coordinates", city: true, place: "Oakland",
			minLon: -122.732, maxLon: -121.824,
			comment: "the ordinary case: a 40km box in the normal frame",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Built the way placeBboxSelect builds it, so the expression under test
			// is the one that ships.
			var inner sq.SelectBuilder
			if tc.city {
				inner = sq.StatementBuilder.
					Select("ne_place.name").
					Column(placeBboxSQL).
					From("ne_10m_populated_places ne_place").
					JoinClause("left join lateral (select ST_Buffer(ne_place.geometry, ?)::geometry as g) ne_geom on true", placeCityRadius).
					Where(sq.Eq{"ne_place.name": tc.place}).
					GroupBy("ne_place.name")
			} else {
				inner = sq.StatementBuilder.
					Select("ne_admin.name").
					Column(placeBboxSQL).
					From("ne_10m_admin_1_states_provinces ne_admin").
					JoinClause("left join lateral (select ne_admin.geometry::geometry as g) ne_geom on true").
					Where(sq.Eq{"ne_admin.name": tc.place}).
					GroupBy("ne_admin.name")
			}
			q := sq.StatementBuilder.
				Select("ST_XMin(t.bbox) as min_lon", "ST_XMax(t.bbox) as max_lon").
				FromSelect(inner, "t")

			var got []struct {
				MinLon float64 `db:"min_lon"`
				MaxLon float64 `db:"max_lon"`
			}
			if err := dbutil.Select(context.Background(), dbx, q, &got); err != nil {
				t.Fatal(err)
			}
			// A kilometre of tolerance: the assertion is about which frame was
			// chosen, not about how finely ST_Buffer approximates a circle.
			if assert.Len(t, got, 1, "expected one row for %s", tc.place) {
				assert.InDelta(t, tc.minLon, got[0].MinLon, 1e-2, tc.comment)
				assert.InDelta(t, tc.maxLon, got[0].MaxLon, 1e-2, tc.comment)
				assert.Less(t, got[0].MaxLon-got[0].MinLon, 180.0, "a usable extent is never wider than half the globe")
			}
		})
	}
}
