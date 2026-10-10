package sync

import (
	"context"
	"testing"
	"time"

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

// oifTestOperator inserts an Atlas operator associated with feed.
func oifTestOperator(t *testing.T, atx tldb.Adapter, feed dmfr.Feed) dmfr.Operator {
	operator := dmfr.Operator{OnestopID: tt.NewString("o-test"), Name: tt.NewString("Test")}
	operator.AssociatedFeeds = dmfr.OperatorAssociatedFeeds{{FeedOnestopID: tt.NewString(feed.FeedID)}}
	operator.ID = testdb.ShouldInsert(t, atx, &operator)
	return operator
}

func TestGetPlaces(t *testing.T) {
	// Places come back in rank order, each once, so a refreshed row holds the
	// same string from one sync to the next.
	ctx := context.Background()
	err := testdb.TempSqlite(func(atx tldb.Adapter) error {
		oifTestFeed(t, atx, "MB")
		agency := gtfs.Agency{}
		testdb.ShouldGet(t, atx, &agency, "select * from gtfs_agencies where agency_id = ?", "MB")
		places := []struct {
			rank float64
			name string
		}{{0.3, "Oakland"}, {0.6, "San Francisco"}, {0.3, "Berkeley"}, {0.3, "Oakland"}, {0.1, "Alameda"}}
		for _, p := range places {
			q := atx.Sqrl().Insert("tl_agency_places").SetMap(map[string]any{
				"feed_version_id": agency.FeedVersionID,
				"agency_id":       agency.ID,
				"rank":            p.rank,
				"name":            p.name,
				"adm1name":        "California",
				"adm0name":        "United States",
			})
			qstr, qargs, err := q.ToSql()
			require.NoError(t, err)
			_, err = atx.DBX().ExecContext(ctx, qstr, qargs...)
			require.NoError(t, err)
		}
		got, err := getPlaces(ctx, atx, agency.ID)
		require.NoError(t, err)
		assert.Equal(t, "San Francisco, California, United States / Oakland, California, United States / Berkeley, California, United States", got)
		return nil
	})
	require.NoError(t, err)
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

func TestUpdateOifs_Rename(t *testing.T) {
	// A matched row picks up its operator's new name after a rename in Atlas.
	ctx := context.Background()
	err := testdb.TempSqlite(func(atx tldb.Adapter) error {
		feed := oifTestFeed(t, atx, "MB")
		operator := oifTestOperator(t, atx, feed)
		prev := dmfr.OperatorAssociatedFeed{OperatorID: tt.NewInt(operator.ID), FeedID: feed.ID, ResolvedGtfsAgencyID: tt.NewString("MB"), ResolvedOnestopID: tt.NewString("o-test"), ResolvedName: tt.NewString("Old name")}
		prevID := testdb.ShouldInsert(t, atx, &prev)

		updated, err := updateOifs(ctx, atx, operator)
		require.NoError(t, err)
		assert.True(t, updated)
		names := []string{}
		testdb.ShouldSelect(t, atx, &names, "select resolved_name from current_operators_in_feed where id = ?", prevID)
		assert.Equal(t, []string{"Test"}, names)

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

func TestDeleteHiddenOifs(t *testing.T) {
	// A soft-deleted operator's row is removed, and the agency it covered gets a
	// generated row in its place.
	ctx := context.Background()
	err := testdb.TempSqlite(func(atx tldb.Adapter) error {
		feed := oifTestFeed(t, atx, "MB")
		operator := dmfr.Operator{OnestopID: tt.NewString("o-gone"), Name: tt.NewString("Gone"), DeletedAt: tt.NewTime(time.Now())}
		operator.ID = testdb.ShouldInsert(t, atx, &operator)
		hidden := dmfr.OperatorAssociatedFeed{OperatorID: tt.NewInt(operator.ID), FeedID: feed.ID, ResolvedGtfsAgencyID: tt.NewString("MB"), ResolvedOnestopID: tt.NewString("o-gone")}
		testdb.ShouldInsert(t, atx, &hidden)

		require.NoError(t, deleteHiddenOifs(ctx, atx))
		_, err := feedUpdateOifs(ctx, atx, feed)
		require.NoError(t, err)
		rows := []dmfr.OperatorAssociatedFeed{}
		testdb.ShouldSelect(t, atx, &rows, "select * from current_operators_in_feed where feed_id = ?", feed.ID)
		if assert.Len(t, rows, 1) {
			assert.False(t, rows[0].OperatorID.Valid, "a generated row")
			assert.Equal(t, "MB", rows[0].ResolvedGtfsAgencyID.Val)
		}
		return nil
	})
	require.NoError(t, err)
}
