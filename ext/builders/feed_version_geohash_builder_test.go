package builders

import (
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/adapters"
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
		_, writer, err := newMockCopier(testreader.ExampleFeedBART.URL, b)
		if err != nil {
			t.Fatal(err)
		}
		var p3, p5 []*FeedVersionGeohash
		for _, ent := range writer.Reader.OtherList {
			if v, ok := ent.(*FeedVersionGeohash); ok {
				switch len(v.Geohash.Val) {
				case 3:
					p3 = append(p3, v)
				case 5:
					p5 = append(p5, v)
				}
			}
		}
		assert.NotEmpty(t, p3, "p3 cells emitted")
		assert.NotEmpty(t, p5, "p5 cells emitted")
		assert.Greater(t, len(p5), len(p3),
			"p5 should produce more cells than p3 for the same stops")
		// BART is in the SF Bay area — geohash prefix "9q"
		for _, c := range p3 {
			assert.True(t, strings.HasPrefix(c.Geohash.Val, "9q"),
				"BART p3 cell should be in 9q* range, got %q", c.Geohash.Val)
			assert.Positive(t, c.StopCount.Val)
		}
		// Total stop_count across p3 cells equals total across p5 cells
		// (each valid stop contributes 1 to exactly one cell at each precision)
		var sumP3, sumP5 int64
		for _, c := range p3 {
			sumP3 += c.StopCount.Val
		}
		for _, c := range p5 {
			sumP5 += c.StopCount.Val
		}
		assert.Equal(t, sumP3, sumP5,
			"total stop counts at p3 and p5 must match (same stops, different bucketing)")
	})

	t.Run("flex feed emits zero-count cells from location polygons", func(t *testing.T) {
		b := NewFeedVersionGeohashBuilder()
		_, writer, err := newMockCopier(testpath.RelPath("testdata/gtfs-external/ctran-flex.zip"), b)
		if err != nil {
			t.Fatal(err)
		}
		var total, zeroCount int
		for _, ent := range writer.Reader.OtherList {
			if v, ok := ent.(*FeedVersionGeohash); ok {
				total++
				if v.StopCount.Val == 0 {
					zeroCount++
				}
			}
		}
		assert.Positive(t, total, "flex feed should emit cells")
		assert.Positive(t, zeroCount,
			"flex feed should emit at least one stop_count=0 cell covering its location polygons")
	})

	t.Run("filters bad coordinates", func(t *testing.T) {
		b := NewFeedVersionGeohashBuilder()
		stops := []struct {
			id       string
			lon, lat float64
		}{
			{"good_sf", -122.4, 37.8},      // valid, San Francisco
			{"good_tx", -97.49, 25.93},     // valid, Brownsville
			{"null_island", 0, 0},          // bad
			{"lon_out_of_range", 200, 37},  // bad
			{"lat_out_of_range", -122, 95}, // bad
		}
		for _, s := range stops {
			st := &gtfs.Stop{StopID: tt.NewString(s.id)}
			st.SetCoordinates([2]float64{s.lon, s.lat})
			if err := b.AfterWrite(s.id, st, nil); err != nil {
				t.Fatal(err)
			}
		}
		if err := b.Copy(&recordingEntityCopier{}); err != nil {
			t.Fatal(err)
		}
		cells := b.Cells()
		// 2 valid stops in different metros → 2 distinct p3 + 2 distinct p5 = 4 entries
		assert.Len(t, cells, 4, "got cells %v", cells)
		// Each cell should have stop_count = 1 (no two valid stops share a cell)
		for c, n := range cells {
			assert.EqualValues(t, 1, n, "cell %s should have count 1", c)
		}
		// Null island cell must be absent at both precisions
		assert.NotContains(t, cells, geohash.EncodeWithPrecision(0, 0, 3))
		assert.NotContains(t, cells, geohash.EncodeWithPrecision(0, 0, 5))
	})
}

// recordingEntityCopier is a minimal EntityCopier that captures emitted
// entities, for unit-testing builder Copy() output without a full pipeline.
type recordingEntityCopier struct {
	emitted []tt.Entity
}

func (rc *recordingEntityCopier) CopyEntity(ent tt.Entity) error {
	rc.emitted = append(rc.emitted, ent)
	return nil
}

func (rc *recordingEntityCopier) CopyEntities(ents []tt.Entity) error {
	rc.emitted = append(rc.emitted, ents...)
	return nil
}

func (rc *recordingEntityCopier) Reader() adapters.Reader { return nil }
func (rc *recordingEntityCopier) Writer() adapters.Writer { return nil }
