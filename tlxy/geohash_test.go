package tlxy

import (
	"math"
	"testing"

	"github.com/mmcloughlin/geohash"
	"github.com/stretchr/testify/assert"
)

func TestIsValidStopCoord(t *testing.T) {
	cases := []struct {
		name     string
		lon, lat float64
		valid    bool
	}{
		{"san francisco", -122.4, 37.8, true},
		{"null island", 0, 0, false},
		{"longitude only zero", 0, 37.8, true},
		{"latitude only zero", -122.4, 0, true},
		{"north pole", 0, 90, true},
		{"south pole", 0, -90, true},
		{"antimeridian east", 180, 0, true},
		{"antimeridian west", -180, 0, true},
		{"lon out of range positive", 181, 0, false},
		{"lon out of range negative", -181, 0, false},
		{"lat out of range positive", 0, 91, false},
		{"lat out of range negative", 0, -91, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.valid, IsValidStopCoord(c.lon, c.lat))
		})
	}
}

func TestGeohashCellSize(t *testing.T) {
	// Expected sizes from the geohash reference table: square at odd precisions
	// (lat/lon bits split evenly), 2:1 wide-rectangle at even precisions
	// (extra bit goes to lon).
	cases := []struct {
		precision            uint
		expectLon, expectLat float64
	}{
		{1, 45.0, 45.0},            // 360/8, 180/4
		{3, 1.40625, 1.40625},      // 360/256, 180/128
		{4, 0.3515625, 0.17578125}, // 360/1024, 180/1024
		{5, 0.0439453125, 0.0439453125},
	}
	for _, c := range cases {
		lonStep, latStep := geohashCellSize(c.precision)
		if math.Abs(lonStep-c.expectLon) > 1e-9 {
			t.Errorf("p%d lonStep = %f, want %f", c.precision, lonStep, c.expectLon)
		}
		if math.Abs(latStep-c.expectLat) > 1e-9 {
			t.Errorf("p%d latStep = %f, want %f", c.precision, latStep, c.expectLat)
		}
	}
}

func TestCellsCoveringBbox(t *testing.T) {
	t.Run("tiny bbox inside one p3 cell", func(t *testing.T) {
		// ~100m square in San Francisco; sits inside a single p3 cell.
		bbox := BoundingBox{MinLon: -122.42, MinLat: 37.77, MaxLon: -122.419, MaxLat: 37.771}
		cells := CellsCoveringBbox(bbox, 3)
		assert.Len(t, cells, 1)
		// The single cell must contain the bbox corners
		expected := geohash.EncodeWithPrecision(37.77, -122.42, 3)
		assert.Equal(t, []string{expected}, cells)
	})

	t.Run("bbox spanning multiple p3 cells", func(t *testing.T) {
		// ~3° × 3° bbox covers 2-3 cells per axis at p3 (1.4° cells).
		bbox := BoundingBox{MinLon: -123, MinLat: 36, MaxLon: -120, MaxLat: 39}
		cells := CellsCoveringBbox(bbox, 3)
		assert.GreaterOrEqual(t, len(cells), 4)
		assert.LessOrEqual(t, len(cells), 16)
		// Every cell in the result must actually intersect the bbox
		for _, c := range cells {
			b := geohash.BoundingBox(c)
			assert.LessOrEqual(t, b.MinLat, bbox.MaxLat, "cell %s south edge above bbox", c)
			assert.GreaterOrEqual(t, b.MaxLat, bbox.MinLat, "cell %s north edge below bbox", c)
			assert.LessOrEqual(t, b.MinLng, bbox.MaxLon, "cell %s west edge past bbox", c)
			assert.GreaterOrEqual(t, b.MaxLng, bbox.MinLon, "cell %s east edge before bbox", c)
		}
	})

	t.Run("p5 covers many more cells than p3 for same bbox", func(t *testing.T) {
		bbox := BoundingBox{MinLon: -122.5, MinLat: 37.5, MaxLon: -122.0, MaxLat: 38.0}
		p3Cells := CellsCoveringBbox(bbox, 3)
		p5Cells := CellsCoveringBbox(bbox, 5)
		assert.Greater(t, len(p5Cells), len(p3Cells)*10,
			"p5 should yield many more cells than p3 for the same bbox")
	})

	t.Run("returns sorted output", func(t *testing.T) {
		bbox := BoundingBox{MinLon: -123, MinLat: 36, MaxLon: -120, MaxLat: 39}
		cells := CellsCoveringBbox(bbox, 3)
		for i := 1; i < len(cells); i++ {
			assert.Less(t, cells[i-1], cells[i], "cells must be sorted")
		}
	})

	t.Run("precision 0 returns nil", func(t *testing.T) {
		bbox := BoundingBox{MinLon: -122, MinLat: 37, MaxLon: -121, MaxLat: 38}
		assert.Nil(t, CellsCoveringBbox(bbox, 0))
	})

	t.Run("BboxFromFlatCoords returns min/max across pairs", func(t *testing.T) {
		coords := []float64{-122.4, 37.8, -122.5, 37.9, -122.3, 37.7}
		bbox, ok := BboxFromFlatCoords(coords)
		assert.True(t, ok)
		assert.Equal(t, -122.5, bbox.MinLon)
		assert.Equal(t, 37.7, bbox.MinLat)
		assert.Equal(t, -122.3, bbox.MaxLon)
		assert.Equal(t, 37.9, bbox.MaxLat)
	})

	t.Run("BboxFromFlatCoords empty input returns false", func(t *testing.T) {
		_, ok := BboxFromFlatCoords(nil)
		assert.False(t, ok)
	})

	t.Run("BboxFromPointRadius widens at equator, narrows at poles", func(t *testing.T) {
		// 10 km at (0, 0): latDelta ~ 10000/111320 ~ 0.0898 deg in each dimension
		eq := BboxFromPointRadius(0, 0, 10_000)
		assert.InDelta(t, -0.0898, eq.MinLon, 0.001)
		assert.InDelta(t, 0.0898, eq.MaxLon, 0.001)
		assert.InDelta(t, -0.0898, eq.MinLat, 0.001)
		assert.InDelta(t, 0.0898, eq.MaxLat, 0.001)

		// Same radius at 60°N: lon span doubles (cos 60° = 0.5), lat unchanged
		mid := BboxFromPointRadius(0, 60, 10_000)
		assert.InDelta(t, 0.1797, mid.MaxLon, 0.001)
		assert.InDelta(t, 0.0898, mid.MaxLat-60, 0.001)
	})

	t.Run("BboxFromPointRadius caps lon at poles", func(t *testing.T) {
		// At lat=89.99, cos is ~tiny; lon delta would explode without the cap
		polar := BboxFromPointRadius(0, 89.99, 10_000)
		assert.Equal(t, -180.0, polar.MinLon)
		assert.Equal(t, 180.0, polar.MaxLon)
	})

	t.Run("Brownsville bbox excludes Seattle p3 cell", func(t *testing.T) {
		// Verify the discrimination property: a feed with stops only around
		// Brownsville TX must not produce p3 cells that overlap a Seattle bbox.
		seattle := BoundingBox{MinLon: -122.45, MinLat: 47.5, MaxLon: -122.2, MaxLat: 47.7}
		seattleCells := CellsCoveringBbox(seattle, 3)

		brownsvilleHash := geohash.EncodeWithPrecision(25.93, -97.50, 3)
		nullIslandHash := geohash.EncodeWithPrecision(0, 0, 3)
		myanmarHash := geohash.EncodeWithPrecision(25.91, 97.49, 3)

		for _, c := range seattleCells {
			assert.NotEqual(t, brownsvilleHash, c)
			assert.NotEqual(t, nullIslandHash, c)
			assert.NotEqual(t, myanmarHash, c)
		}
	})
}
