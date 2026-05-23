package gql

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/stretchr/testify/assert"
)

// Flipping feed_states.public = false strips the only access path the default
// empty PermFilter has, so every loader under test should now return zero rows.
// If pfJoinCheckFv is missing from any loader, that loader will keep returning
// data and the corresponding subtest fails.
func TestPermFilter_Negative(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{}, func(cfg model.Config) {
		ctx := model.WithConfig(context.Background(), cfg)
		if _, err := cfg.Finder.DBX().ExecContext(ctx, "update feed_states set public = false"); err != nil {
			t.Fatal(err)
		}

		const haFv = 6
		segmentIDs := []int{1418704, 1418711}
		routeIDsWithSegments := []int{34, 39}

		t.Run("SegmentsByFeedVersionIDs", func(t *testing.T) {
			got, err := cfg.Finder.SegmentsByFeedVersionIDs(ctx, nil, nil, []int{haFv})
			assert.NoError(t, err)
			if assert.Len(t, got, 1) {
				assert.Empty(t, got[0], "expected no segments when caller cannot see FV %d", haFv)
			}
		})

		t.Run("SegmentsByIDs", func(t *testing.T) {
			ents, errs := cfg.Finder.SegmentsByIDs(ctx, segmentIDs)
			for _, err := range errs {
				assert.NoError(t, err)
			}
			if assert.Len(t, ents, len(segmentIDs)) {
				for i, ent := range ents {
					assert.Nil(t, ent, "expected segment %d (id %d) to be nil", i, segmentIDs[i])
				}
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
			got, err := cfg.Finder.SegmentPatternsBySegmentIDs(ctx, nil, nil, segmentIDs)
			assert.NoError(t, err)
			if assert.Len(t, got, len(segmentIDs)) {
				for i, group := range got {
					assert.Empty(t, group, "expected no segment patterns for segment id %d", segmentIDs[i])
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
