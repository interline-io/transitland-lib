package dbfinder

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/tldb/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A stop can be in several location groups, and stops in one batch can share a group.
// Each stop gets each of its groups once, and each group each of its stops once.
func TestFinder_SharedLocationGroupStops(t *testing.T) {
	ctx := context.Background()

	// Runs in a transaction that is always rolled back: the test adds a group membership,
	// and the test database is shared with the other server packages.
	db := testutil.MustOpenTestDB(t)
	tx, err := db.Beginx()
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })
	f := NewFinder(dbutil.WithQueryLogger(tx, false, 0))

	// C-TRAN flex, from the standard test fixtures.
	sha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	fvs, err := f.FindFeedVersions(ctx, nil, nil, nil, &model.FeedVersionFilter{Sha1: &sha1})
	require.NoError(t, err)
	require.Len(t, fvs, 1, "expected the ctran-flex test feed version")
	fvid := fvs[0].ID
	groupID := func(locationGroupID string) int {
		t.Helper()
		groups, err := f.LocationGroupsByFeedVersionIDs(ctx, nil, &model.LocationGroupFilter{LocationGroupID: &locationGroupID}, []int{fvid})
		require.NoError(t, err)
		require.Len(t, groups[0], 1, "expected location group %s", locationGroupID)
		return groups[0][0].ID
	}
	fairgrounds := groupID("location_group_id__138b146e-30ff-4837-baf8-bd75b47bac6a")
	vaMedicalCenter := groupID("location_group_id__db7489d3-7478-4d3b-a47f-60c58e3fed6e")

	// Two of the fairgrounds stops.
	stops, errs := f.StopsByFeedVersionStopIDs(ctx, []model.FVEntityID{
		{FeedVersionID: fvid, EntityID: "stop_id__756c0e65-32d2-4e32-a6b7-a15c3c22e6cf"},
		{FeedVersionID: fvid, EntityID: "stop_id__2e44e463-310b-4069-a709-fa0eb8f73ba9"},
	})
	require.Nil(t, errs)
	require.Len(t, stops, 2)
	require.NotNil(t, stops[0], "expected the first fairgrounds stop")
	require.NotNil(t, stops[1], "expected the second fairgrounds stop")
	shared, other := stops[0].ID, stops[1].ID

	// The first one joins the VA Medical Center group too.
	membership := gtfs.LocationGroupStop{}
	membership.FeedVersionID = fvid
	membership.LocationGroupID.SetInt(vaMedicalCenter)
	membership.StopID.SetInt(shared)
	_, err = postgres.NewPostgresAdapterFromDBX(tx).Insert(ctx, &membership)
	require.NoError(t, err)

	t.Run("location groups by stop", func(t *testing.T) {
		groups, err := f.LocationGroupsByStopIDs(ctx, nil, []int{shared, other})
		require.NoError(t, err)
		require.Len(t, groups, 2)
		groupIDs := func(ents []*model.LocationGroup) []int {
			var ret []int
			for _, ent := range ents {
				ret = append(ret, ent.ID)
			}
			return ret
		}
		assert.ElementsMatch(t, []int{fairgrounds, vaMedicalCenter}, groupIDs(groups[0]))
		assert.ElementsMatch(t, []int{fairgrounds}, groupIDs(groups[1]))
	})

	t.Run("stops by location group", func(t *testing.T) {
		groupStops, err := f.StopsByLocationGroupIDs(ctx, nil, []int{fairgrounds, vaMedicalCenter})
		require.NoError(t, err)
		require.Len(t, groupStops, 2)
		count := func(ents []*model.Stop, stopID int) int {
			n := 0
			for _, ent := range ents {
				if ent.ID == stopID {
					n++
				}
			}
			return n
		}
		assert.Equal(t, 1, count(groupStops[0], shared), "fairgrounds should list the shared stop once")
		assert.Equal(t, 1, count(groupStops[0], other), "fairgrounds should list the other stop once")
		assert.Equal(t, 1, count(groupStops[1], shared), "VA Medical Center should list the shared stop once")
		assert.Equal(t, 0, count(groupStops[1], other), "VA Medical Center should not list the other stop")
	})
}
