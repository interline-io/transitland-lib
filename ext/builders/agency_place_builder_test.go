package builders

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/adapters/direct"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/internal/testreader"
	"github.com/interline-io/transitland-lib/tlcsv"
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
	e := NewAgencyPlaceBuilder()
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
