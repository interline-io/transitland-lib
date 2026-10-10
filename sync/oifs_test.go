package sync

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// oifTestFeed inserts a feed whose materialized version holds one agency for each
// agency_id given, an empty one for an agency without an agency_id.
func oifTestFeed(t *testing.T, atx tldb.Adapter, agencyIDs ...string) dmfr.Feed {
	fv := testdb.CreateTestFeedVersion(atx, "")
	fs := dmfr.FeedState{FeedID: fv.FeedID, MaterializedFeedVersionID: tt.NewInt(fv.ID)}
	testdb.ShouldInsert(t, atx, &fs)
	for _, id := range agencyIDs {
		agency := gtfs.Agency{AgencyID: tt.NewString(id), AgencyName: tt.NewString("Agency " + id)}
		agency.FeedVersionID = fv.ID
		testdb.ShouldInsert(t, atx, &agency)
	}
	feed := dmfr.Feed{}
	testdb.ShouldGet(t, atx, &feed, "select * from current_feeds where id = ?", fv.FeedID)
	return feed
}

func oifTestOperator(t *testing.T, atx tldb.Adapter, feed dmfr.Feed) dmfr.Operator {
	operator := dmfr.Operator{OnestopID: tt.NewString("o-test"), Name: tt.NewString("Test")}
	operator.AssociatedFeeds = dmfr.OperatorAssociatedFeeds{{FeedOnestopID: tt.NewString(feed.FeedID)}}
	operator.ID = testdb.ShouldInsert(t, atx, &operator)
	return operator
}

func TestUpdateOifs_NullAgencyID(t *testing.T) {
	// An association written before its feed had a version holds a NULL resolved
	// agency id. Once the feed's only agency, which has no agency_id, resolves it
	// to '', the row is rewritten in place, and a later sync changes nothing.
	ctx := context.Background()
	err := testdb.TempSqlite(func(atx tldb.Adapter) error {
		feed := oifTestFeed(t, atx, "")
		operator := oifTestOperator(t, atx, feed)
		prev := dmfr.OperatorAssociatedFeed{OperatorID: tt.NewInt(operator.ID), FeedID: feed.ID, ResolvedOnestopID: tt.NewString("o-test")}
		prevID := testdb.ShouldInsert(t, atx, &prev)

		updated, err := updateOifs(ctx, atx, operator)
		require.NoError(t, err)
		assert.True(t, updated)
		count := 0
		testdb.ShouldGet(t, atx, &count, "select count(*) from current_operators_in_feed where id = ? and resolved_gtfs_agency_id = ''", prevID)
		assert.Equal(t, 1, count, "the row now holds ''")
		testdb.ShouldGet(t, atx, &count, "select count(*) from current_operators_in_feed where operator_id = ?", operator.ID)
		assert.Equal(t, 1, count, "rewritten rather than replaced")

		updated, err = updateOifs(ctx, atx, operator)
		require.NoError(t, err)
		assert.False(t, updated, "a second sync finds nothing to change")
		return nil
	})
	require.NoError(t, err)
}

func TestFeedUpdateOifs_GeneratedRowBesideAtlasRow(t *testing.T) {
	// A generated row left beside an Atlas row for the same agency is deleted and
	// the Atlas row kept. The generated row is inserted second, so it is also the
	// one the database returns last.
	ctx := context.Background()
	err := testdb.TempSqlite(func(atx tldb.Adapter) error {
		feed := oifTestFeed(t, atx, "MB")
		operator := oifTestOperator(t, atx, feed)
		atlas := dmfr.OperatorAssociatedFeed{OperatorID: tt.NewInt(operator.ID), FeedID: feed.ID, ResolvedGtfsAgencyID: tt.NewString("MB"), ResolvedOnestopID: tt.NewString("o-test")}
		atlasID := testdb.ShouldInsert(t, atx, &atlas)
		generated := dmfr.OperatorAssociatedFeed{FeedID: feed.ID, ResolvedGtfsAgencyID: tt.NewString("MB"), ResolvedOnestopID: tt.NewString("o-test")}
		testdb.ShouldInsert(t, atx, &generated)

		updated, err := feedUpdateOifs(ctx, atx, feed)
		require.NoError(t, err)
		assert.True(t, updated)
		ids := []int{}
		testdb.ShouldSelect(t, atx, &ids, "select id from current_operators_in_feed where feed_id = ?", feed.ID)
		assert.Equal(t, []int{atlasID}, ids)
		return nil
	})
	require.NoError(t, err)
}
