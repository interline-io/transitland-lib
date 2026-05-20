package builders

import (
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/internal/testreader"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/mmcloughlin/geohash"
	"github.com/stretchr/testify/assert"
	geom "github.com/twpayne/go-geom"
)

func TestFeedVersionGeohashBuilder(t *testing.T) {
	t.Run("BART produces p3 cells with positive stop counts", func(t *testing.T) {
		b := NewFeedVersionGeohashBuilder()
		if _, _, err := newMockCopier(testreader.ExampleFeedBART.URL, b); err != nil {
			t.Fatal(err)
		}
		cells := b.Cells()
		assert.NotEmpty(t, cells, "cells emitted")
		for c, n := range cells {
			// Only p3 (length 3) cells are computed for now.
			assert.Len(t, c, 3, "expected only p3 cells, got %q", c)
			// BART is in the SF Bay area — geohash prefix "9q"
			assert.True(t, strings.HasPrefix(c, "9q"),
				"BART p3 cell should be in 9q* range, got %q", c)
			assert.Positive(t, n)
		}
	})

	t.Run("flex location polygon emits a zero-count cell where no stops exist", func(t *testing.T) {
		b := NewFeedVersionGeohashBuilder()
		// A fixed stop in San Francisco.
		sf := &gtfs.Stop{StopID: tt.NewString("sf")}
		sf.SetCoordinates([2]float64{-122.4, 37.8})
		if err := b.AfterWrite("sf", sf, nil); err != nil {
			t.Fatal(err)
		}
		// A flex location polygon far away in south Texas — a different p3 cell
		// with no fixed stops, so it must surface as a zero-count cell.
		ring := []float64{-97.55, 25.85, -97.45, 25.85, -97.45, 25.95, -97.55, 25.95, -97.55, 25.85}
		poly := geom.NewPolygonFlat(geom.XY, ring, []int{len(ring)})
		loc := &gtfs.Location{Geometry: tt.NewGeometry(poly)}
		if err := b.AfterWrite("loc", loc, nil); err != nil {
			t.Fatal(err)
		}
		if err := b.Copy(nil); err != nil {
			t.Fatal(err)
		}
		cells := b.Cells()
		sfCell := geohash.EncodeWithPrecision(37.8, -122.4, 3)
		txCell := geohash.EncodeWithPrecision(25.9, -97.5, 3)
		assert.NotEqual(t, sfCell, txCell, "test setup: SF and TX must fall in different p3 cells")
		assert.Equal(t, 1, cells[sfCell], "stop cell should have a positive count")
		assert.Contains(t, cells, txCell, "flex polygon cell should be present")
		assert.Equal(t, 0, cells[txCell], "flex-only cell should have stop_count 0")
	})

	t.Run("keeps all coordinates including bad ones", func(t *testing.T) {
		b := NewFeedVersionGeohashBuilder()
		stops := []struct {
			id       string
			lon, lat float64
		}{
			{"good_sf", -122.4, 37.8},  // San Francisco
			{"good_tx", -97.49, 25.93}, // Brownsville
			{"null_island", 0, 0},      // bad data, but kept
		}
		for _, s := range stops {
			st := &gtfs.Stop{StopID: tt.NewString(s.id)}
			st.SetCoordinates([2]float64{s.lon, s.lat})
			if err := b.AfterWrite(s.id, st, nil); err != nil {
				t.Fatal(err)
			}
		}
		if err := b.Copy(nil); err != nil {
			t.Fatal(err)
		}
		cells := b.Cells()
		// 3 stops in different cells → 3 distinct p3 entries
		assert.Len(t, cells, 3, "got cells %v", cells)
		for c, n := range cells {
			assert.EqualValues(t, 1, n, "cell %s should have count 1", c)
		}
		// Null island is retained, so bad-coordinate stops remain discoverable
		// via the cells.
		assert.Contains(t, cells, geohash.EncodeWithPrecision(0, 0, 3))
	})
}
