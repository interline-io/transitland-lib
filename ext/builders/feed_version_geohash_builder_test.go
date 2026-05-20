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
	t.Run("BART produces p3 and p5 cells with positive stop counts", func(t *testing.T) {
		b := NewFeedVersionGeohashBuilder()
		if _, _, err := newMockCopier(testreader.ExampleFeedBART.URL, b); err != nil {
			t.Fatal(err)
		}
		cells := b.Cells()
		assert.NotEmpty(t, cells, "cells emitted")
		var p3, p5 int
		for c, n := range cells {
			// Stops are encoded at p3 (discovery) and p5 (fingerprint).
			assert.True(t, len(c) == 3 || len(c) == 5, "expected p3 or p5 cell, got %q", c)
			switch len(c) {
			case 3:
				p3++
			case 5:
				p5++
			}
			// BART is in the SF Bay area — geohash prefix "9q" (p5 extends the p3 prefix).
			assert.True(t, strings.HasPrefix(c, "9q"),
				"BART cell should be in 9q* range, got %q", c)
			assert.Positive(t, n)
		}
		assert.Positive(t, p3, "expected p3 cells")
		assert.Positive(t, p5, "expected p5 cells")
		assert.GreaterOrEqual(t, p5, p3, "p5 cells are at least as fine-grained as p3")
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
		// Flex cells are emitted at every precision, so the zone also surfaces a
		// p5 cell with no stop.
		txCellP5 := geohash.EncodeWithPrecision(25.9, -97.5, 5)
		assert.Contains(t, cells, txCellP5, "flex polygon should also emit a p5 cell")
		assert.Equal(t, 0, cells[txCellP5], "flex-only p5 cell should have stop_count 0")
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
		// 3 stops in distinct cells → 3 p3 + 3 p5 = 6 distinct entries
		assert.Len(t, cells, 6, "got cells %v", cells)
		for c, n := range cells {
			assert.EqualValues(t, 1, n, "cell %s should have count 1", c)
		}
		// Null island is retained at both precisions, so bad-coordinate stops
		// remain discoverable via the cells.
		assert.Contains(t, cells, geohash.EncodeWithPrecision(0, 0, 3))
		assert.Contains(t, cells, geohash.EncodeWithPrecision(0, 0, 5))
	})

	t.Run("skips stops with no coordinate", func(t *testing.T) {
		b := NewFeedVersionGeohashBuilder()
		// A stop with a real coordinate.
		good := &gtfs.Stop{StopID: tt.NewString("good")}
		good.SetCoordinates([2]float64{-122.4, 37.8})
		if err := b.AfterWrite("good", good, nil); err != nil {
			t.Fatal(err)
		}
		// A stop with no coordinate at all (Geometry.Valid == false) must not add
		// a (0,0) Null Island cell.
		noCoord := &gtfs.Stop{StopID: tt.NewString("nocoord")}
		if err := b.AfterWrite("nocoord", noCoord, nil); err != nil {
			t.Fatal(err)
		}
		if err := b.Copy(nil); err != nil {
			t.Fatal(err)
		}
		cells := b.Cells()
		assert.NotContains(t, cells, geohash.EncodeWithPrecision(0, 0, 3),
			"missing-coordinate stop must not add a null-island cell")
		assert.NotContains(t, cells, geohash.EncodeWithPrecision(0, 0, 5))
		// Only the good stop's p3 + p5 cells remain.
		assert.Len(t, cells, 2, "got cells %v", cells)
	})
}
