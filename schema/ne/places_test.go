package ne

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/stretchr/testify/assert"
	"github.com/twpayne/go-geom"
)

// rect returns the polygon of a lon/lat box.
func rect(minLon, minLat, maxLon, maxLat float64) *geom.Polygon {
	return geom.NewPolygonFlat(geom.XY, []float64{minLon, minLat, maxLon, minLat, maxLon, maxLat, minLon, maxLat, minLon, minLat}, []int{10})
}

func TestPlaceIndex(t *testing.T) {
	// Two areas side by side, split at the prime meridian. Westville's source
	// names the wrong area, and Seaport lies in neither.
	west := AdminArea{Name: "West", Admin: "Country", IsoA2: "CC", Iso31662: "CC-W", Geometry: rect(-1, 0, 0, 1)}
	east := AdminArea{Name: "East", Admin: "Country", IsoA2: "CC", Iso31662: "CC-E", Geometry: geom.NewMultiPolygon(geom.XY).MustSetCoords([][][]geom.Coord{rect(0, 0, 1, 1).Coords()})}
	ix, err := NewPlaceIndex([]AdminArea{west, east}, []PopulatedPlace{
		{Name: "Westville", Adm1Name: "East", Adm0Name: "Country", Point: tlxy.Point{Lon: -0.5, Lat: 0.5}},
		{Name: "Eastville", Adm1Name: "East", Adm0Name: "Country", Point: tlxy.Point{Lon: 0.5, Lat: 0.5}},
		{Name: "Seaport", Adm1Name: "Coast", Adm0Name: "Country", Point: tlxy.Point{Lon: 0.5, Lat: 1.5}},
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("admin holding a point", func(t *testing.T) {
		a, ok := ix.Admin(tlxy.Point{Lon: -0.5, Lat: 0.5})
		assert.True(t, ok)
		assert.Equal(t, "West", a.Name)
		a, ok = ix.Admin(tlxy.Point{Lon: 0.5, Lat: 0.5})
		assert.True(t, ok)
		assert.Equal(t, "East", a.Name)
		_, ok = ix.Admin(tlxy.Point{Lon: 0.5, Lat: 1.1})
		assert.False(t, ok, "a point outside every area has no admin")
	})

	t.Run("nearest admin", func(t *testing.T) {
		a, ok := ix.NearestAdmin(tlxy.Point{Lon: 0.5, Lat: 1.1})
		assert.True(t, ok, "an area within a quarter degree")
		assert.Equal(t, "CC-E", a.Iso31662)
		_, ok = ix.NearestAdmin(tlxy.Point{Lon: 0.5, Lat: 2})
		assert.False(t, ok)
	})

	t.Run("nearest place", func(t *testing.T) {
		// Westville is 11 km away, and takes its area's names.
		p, ok := ix.NearestPlace(tlxy.Point{Lon: -0.4, Lat: 0.5}, 40000)
		assert.True(t, ok)
		assert.Equal(t, "Westville", p.Name)
		assert.Equal(t, "West", p.Adm1Name)
		// Seaport lies in no area, so it keeps its own names.
		p, ok = ix.NearestPlace(tlxy.Point{Lon: 0.5, Lat: 1.4}, 40000)
		assert.True(t, ok)
		assert.Equal(t, "Seaport", p.Name)
		assert.Equal(t, "Coast", p.Adm1Name)
		// The nearest place, Westville, is 44 km away.
		_, ok = ix.NearestPlace(tlxy.Point{Lon: -0.5, Lat: 0.9}, 40000)
		assert.False(t, ok)
	})
}

func TestPlaces_NearestAdmin(t *testing.T) {
	ix, err := Places()
	if err != nil {
		t.Fatal(err)
	}
	// Natural Earth's 10m boundaries are slightly simplified, as at the
	// Georgia/Florida border here. Postgres matches only the points inside an
	// area, so it doesn't check the one point offshore.
	tcs := []struct {
		name       string
		point      tlxy.Point
		expectAdm0 string
		expectAdm1 string
		expectOk   bool
		skipPg     bool
	}{
		{name: "new york", expectAdm0: "United States of America", expectAdm1: "New York", expectOk: true, point: tlxy.Point{Lon: -74.132285, Lat: 40.625665}},
		{name: "california", expectAdm0: "United States of America", expectAdm1: "California", expectOk: true, point: tlxy.Point{Lon: -122.431297, Lat: 37.773972}},
		{name: "kansas 1", expectAdm0: "United States of America", expectAdm1: "Kansas", expectOk: true, point: tlxy.Point{Lon: -98.85867269364557, Lat: 39.96773433000109}},
		{name: "kansas 2", expectAdm0: "United States of America", expectAdm1: "Kansas", expectOk: true, point: tlxy.Point{Lon: -98.85867269364557, Lat: 39.99901}},
		{name: "nebraska 1", expectAdm0: "United States of America", expectAdm1: "Nebraska", expectOk: true, point: tlxy.Point{Lon: -98.862255, Lat: 40.001587}},
		{name: "nebraska 2", expectAdm0: "United States of America", expectAdm1: "Nebraska", expectOk: true, point: tlxy.Point{Lon: -98.867745, Lat: 40.003185}},
		{name: "utah", expectAdm0: "United States of America", expectAdm1: "Utah", expectOk: true, point: tlxy.Point{Lon: -109.056664, Lat: 40.996479}},
		{name: "colorado", expectAdm0: "United States of America", expectAdm1: "Colorado", expectOk: true, point: tlxy.Point{Lon: -109.045685, Lat: 40.997833}},
		{name: "wyoming", expectAdm0: "United States of America", expectAdm1: "Wyoming", expectOk: true, point: tlxy.Point{Lon: -109.050133, Lat: 41.002209}},
		{name: "north dakota", expectAdm0: "United States of America", expectAdm1: "North Dakota", expectOk: true, point: tlxy.Point{Lon: -100.964531, Lat: 45.946934}},
		{name: "georgia", expectAdm0: "United States of America", expectAdm1: "Georgia", expectOk: true, point: tlxy.Point{Lon: -82.066697, Lat: 30.370054}},
		{name: "florida", expectAdm0: "United States of America", expectAdm1: "Florida", expectOk: true, point: tlxy.Point{Lon: -82.046522, Lat: 30.360419}},
		{name: "saskatchewan", expectAdm0: "Canada", expectAdm1: "Saskatchewan", expectOk: true, point: tlxy.Point{Lon: -102.007904, Lat: 58.269615}},
		{name: "manitoba", expectAdm0: "Canada", expectAdm1: "Manitoba", expectOk: true, point: tlxy.Point{Lon: -101.982025, Lat: 58.269245}},
		{name: "paris", expectAdm0: "France", expectAdm1: "Paris", expectOk: true, point: tlxy.Point{Lon: 2.4729377, Lat: 48.8589143}},
		{name: "texas", expectAdm0: "United States of America", expectAdm1: "Texas", expectOk: true, point: tlxy.Point{Lon: -94.794261, Lat: 29.289210}},
		{name: "texas water 1", skipPg: true, expectAdm0: "United States of America", expectAdm1: "Texas", expectOk: true, point: tlxy.Point{Lon: -94.784667, Lat: 29.286234}},
		{name: "texas water 2", expectOk: false, point: tlxy.Point{Lon: -94.237, Lat: 26.874}},
		{name: "null", expectOk: false, point: tlxy.Point{Lon: 0, Lat: 0}},
	}
	_, hasDB := testutil.CheckTestDB()
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			a, ok := ix.NearestAdmin(tc.point)
			assert.Equal(t, tc.expectOk, ok)
			assert.Equal(t, tc.expectAdm0, a.Admin)
			assert.Equal(t, tc.expectAdm1, a.Name)
			if tc.skipPg || !hasDB {
				return
			}
			var pgCheck []struct {
				Name  string
				Admin string
			}
			q := `select ne.name, ne.admin from ne_10m_admin_1_states_provinces ne where ST_Intersects(ne.geometry::geography, ST_MakePoint($1, $2)::geography)`
			if err := testutil.MustOpenTestDB(t).Select(&pgCheck, q, tc.point.Lon, tc.point.Lat); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.expectOk, len(pgCheck) > 0, "different than postgres")
			for _, ent := range pgCheck {
				assert.Equal(t, tc.expectAdm0, ent.Admin, "different than postgres")
				assert.Equal(t, tc.expectAdm1, ent.Name, "different than postgres")
			}
		})
	}
}
