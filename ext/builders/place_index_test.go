package builders

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/stretchr/testify/assert"
	"github.com/twpayne/go-geom"
)

// rect returns the polygon of a lon/lat box.
func rect(minLon, minLat, maxLon, maxLat float64) *geom.Polygon {
	return geom.NewPolygonFlat(geom.XY, []float64{minLon, minLat, maxLon, minLat, maxLon, maxLat, minLon, maxLat, minLon, minLat}, []int{10})
}

func TestPlaceIndex(t *testing.T) {
	ctx := context.Background()
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
		p, ok, err := ix.Admin(ctx, tlxy.Point{Lon: -0.5, Lat: 0.5})
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "West", p.Adm1name.Val)
		assert.False(t, p.Name.Valid)
		p, ok, _ = ix.Admin(ctx, tlxy.Point{Lon: 0.5, Lat: 0.5})
		assert.True(t, ok)
		assert.Equal(t, "East", p.Adm1name.Val)
		_, ok, _ = ix.Admin(ctx, tlxy.Point{Lon: 0.5, Lat: 1.1})
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
		p, ok, err := ix.NearestPlace(ctx, tlxy.Point{Lon: -0.4, Lat: 0.5}, 40000)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "Westville", p.Name.Val)
		assert.Equal(t, "West", p.Adm1name.Val)
		// Seaport lies in no area, so it keeps its own names.
		p, ok, _ = ix.NearestPlace(ctx, tlxy.Point{Lon: 0.5, Lat: 1.4}, 40000)
		assert.True(t, ok)
		assert.Equal(t, "Seaport", p.Name.Val)
		assert.Equal(t, "Coast", p.Adm1name.Val)
		// The nearest place, Westville, is 44 km away.
		_, ok, _ = ix.NearestPlace(ctx, tlxy.Point{Lon: -0.5, Lat: 0.9}, 40000)
		assert.False(t, ok)
	})
}
