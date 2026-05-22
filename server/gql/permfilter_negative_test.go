package gql

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/stretchr/testify/assert"
)

// Verifies PermFilter blocks unauthorized callers from segment / segment-pattern
// loaders (H6) and from the feed-version-scoped service-level and validation-
// report loaders (H10).
//
// Test fixtures mark every feed's feed_states.public = true, so the default
// DenyAllChecker PermFilter ({}) still passes pfJoinCheckFv. Inside a rolled-
// back tx we flip public = false everywhere, so the only way data comes back
// is via AllowedFeeds/AllowedFeedVersions/IsGlobalAdmin — none of which the
// default empty PermFilter has. Every loader should return empty.
//
// Fixture details:
//   - HA feed_version_id = 6 (sha1 c969...) holds the segments+patterns
//     inserted by testdata/server/test_supplement.pgsql, plus
//     feed_version_service_levels and one tl_validation_reports row.
//   - tl_segments ids: 1418704, 1418711.
//   - tl_segment_patterns ids: 1, 2, 3 on routes 34, 39.
func TestPermFilter_Negative(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{}, func(cfg model.Config) {
		ctx := model.WithConfig(context.Background(), cfg)
		if _, err := cfg.Finder.DBX().ExecContext(ctx, "update feed_states set public = false"); err != nil {
			t.Fatal(err)
		}

		const haFv = 6
		segmentIDs := []int{1418704, 1418711}
		segmentPatternIDs := []int{1, 2, 3}
		routeIDsWithSegments := []int{34, 39}

		t.Run("SegmentsByFeedVersionIDs", func(t *testing.T) {
			got, err := cfg.Finder.SegmentsByFeedVersionIDs(ctx, nil, nil, []int{haFv})
			assert.NoError(t, err)
			if assert.Len(t, got, 1) {
				assert.Empty(t, got[0], "expected no segments when caller cannot see FV %d", haFv)
			}
		})

		t.Run("SegmentsByIDs", func(t *testing.T) {
			ents, _ := cfg.Finder.SegmentsByIDs(ctx, segmentIDs)
			for i, ent := range ents {
				assert.Nil(t, ent, "expected segment %d (id %d) to be nil", i, segmentIDs[i])
			}
		})

		t.Run("SegmentsByRouteIDs", func(t *testing.T) {
			got, err := cfg.Finder.SegmentsByRouteIDs(ctx, nil, nil, routeIDsWithSegments)
			assert.NoError(t, err)
			if assert.Len(t, got, len(routeIDsWithSegments)) {
				for i, group := range got {
					assert.Empty(t, group, "expected no segments for route %d", routeIDsWithSegments[i])
				}
			}
		})

		t.Run("SegmentPatternsByRouteIDs", func(t *testing.T) {
			got, err := cfg.Finder.SegmentPatternsByRouteIDs(ctx, nil, nil, routeIDsWithSegments)
			assert.NoError(t, err)
			if assert.Len(t, got, len(routeIDsWithSegments)) {
				for i, group := range got {
					assert.Empty(t, group, "expected no segment patterns for route %d", routeIDsWithSegments[i])
				}
			}
		})

		t.Run("SegmentPatternsBySegmentIDs", func(t *testing.T) {
			got, err := cfg.Finder.SegmentPatternsBySegmentIDs(ctx, nil, nil, segmentPatternIDs)
			assert.NoError(t, err)
			if assert.Len(t, got, len(segmentPatternIDs)) {
				for i, group := range got {
					assert.Empty(t, group, "expected no segment patterns for segment id %d", segmentPatternIDs[i])
				}
			}
		})

		t.Run("ValidationReportsByFeedVersionIDs", func(t *testing.T) {
			got, err := cfg.Finder.ValidationReportsByFeedVersionIDs(ctx, nil, nil, []int{haFv})
			assert.NoError(t, err)
			if assert.Len(t, got, 1) {
				assert.Empty(t, got[0], "expected no validation reports for FV %d", haFv)
			}
		})

		t.Run("FeedVersionServiceLevelsByFeedVersionIDs", func(t *testing.T) {
			got, err := cfg.Finder.FeedVersionServiceLevelsByFeedVersionIDs(ctx, nil, nil, []int{haFv})
			assert.NoError(t, err)
			if assert.Len(t, got, 1) {
				assert.Empty(t, got[0], "expected no service levels for FV %d", haFv)
			}
		})
	})
}
