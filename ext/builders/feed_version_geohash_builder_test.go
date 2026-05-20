package builders

import (
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/interline-io/transitland-lib/internal/testreader"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/mmcloughlin/geohash"
	"github.com/stretchr/testify/assert"
)

func TestFeedVersionGeohashBuilder(t *testing.T) {
	t.Run("BART produces p3 and p5 cells with positive stop counts", func(t *testing.T) {
		b := NewFeedVersionGeohashBuilder()
		if _, _, err := newMockCopier(testreader.ExampleFeedBART.URL, b); err != nil {
			t.Fatal(err)
		}
		cells := b.Cells()
		var p3, p5 []string
		var sumP3, sumP5 int
		for c, n := range cells {
			switch len(c) {
			case 3:
				p3 = append(p3, c)
				sumP3 += n
				// BART is in the SF Bay area — geohash prefix "9q"
				assert.True(t, strings.HasPrefix(c, "9q"),
					"BART p3 cell should be in 9q* range, got %q", c)
				assert.Positive(t, n)
			case 5:
				p5 = append(p5, c)
				sumP5 += n
			}
		}
		assert.NotEmpty(t, p3, "p3 cells emitted")
		assert.NotEmpty(t, p5, "p5 cells emitted")
		assert.Greater(t, len(p5), len(p3),
			"p5 should produce more cells than p3 for the same stops")
		// Total stop_count across p3 cells equals total across p5 cells
		// (each stop contributes 1 to exactly one cell at each precision)
		assert.Equal(t, sumP3, sumP5,
			"total stop counts at p3 and p5 must match (same stops, different bucketing)")
	})

	t.Run("flex feed emits zero-count cells from location polygons", func(t *testing.T) {
		b := NewFeedVersionGeohashBuilder()
		if _, _, err := newMockCopier(testpath.RelPath("testdata/gtfs-external/ctran-flex.zip"), b); err != nil {
			t.Fatal(err)
		}
		var total, zeroCount int
		for _, n := range b.Cells() {
			total++
			if n == 0 {
				zeroCount++
			}
		}
		assert.Positive(t, total, "flex feed should emit cells")
		assert.Positive(t, zeroCount,
			"flex feed should emit at least one stop_count=0 cell covering its location polygons")
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
		// 3 stops in different cells → 3 distinct p3 + 3 distinct p5 = 6 entries
		assert.Len(t, cells, 6, "got cells %v", cells)
		for c, n := range cells {
			assert.EqualValues(t, 1, n, "cell %s should have count 1", c)
		}
		// Null island is retained at both precisions, so bad-coordinate stops
		// remain discoverable via the cells.
		assert.Contains(t, cells, geohash.EncodeWithPrecision(0, 0, 3))
		assert.Contains(t, cells, geohash.EncodeWithPrecision(0, 0, 5))
	})
}
