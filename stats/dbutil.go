package stats

import (
	"context"
	"database/sql"

	sq "github.com/irees/squirrel"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/tldb"
)

func FeedVersionTableDelete(ctx context.Context, atx tldb.Adapter, table string, fvid int, ifExists bool) error {
	// check if table exists before proceeding
	if ifExists {
		ok, err := atx.TableExists(table)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}
	where := sq.Eq{"feed_version_id": fvid}
	_, err := atx.Sqrl().Delete(table).Where(where).ExecContext(ctx)
	if err != nil {
		return err
	}
	return nil
}

func GetFeedState(ctx context.Context, atx tldb.Adapter, feedId int) (dmfr.FeedState, error) {
	// Get state, create if necessary
	fs := dmfr.FeedState{FeedID: feedId}
	if err := atx.Get(ctx, &fs, `SELECT * FROM feed_states WHERE feed_id = ?`, feedId); err == sql.ErrNoRows {
		fs.ID, err = atx.Insert(ctx, &fs)
		if err != nil {
			return fs, err
		}
	} else if err != nil {
		return fs, err
	}
	return fs, nil
}

// UpdateFeedStatePublic creates or updates a feed state's public flag.
// For new feeds (isNew=true): default to public=true unless setPublic is explicitly false
// For existing feeds: only update public flag if setPublic is explicitly set (non-nil)
func UpdateFeedStatePublic(ctx context.Context, atx tldb.Adapter, feedId int, isNew bool, setPublic *bool) (dmfr.FeedState, error) {
	fs := dmfr.FeedState{FeedID: feedId}
	if err := atx.Get(ctx, &fs, `SELECT * FROM feed_states WHERE feed_id = ?`, feedId); err == sql.ErrNoRows {
		// New feed state - default to public unless explicitly set to private
		if setPublic != nil {
			fs.Public = *setPublic
		} else {
			fs.Public = true // Default: new feeds are public
		}
		fs.ID, err = atx.Insert(ctx, &fs)
		if err != nil {
			return fs, err
		}
	} else if err != nil {
		return fs, err
	} else {
		// Existing feed state - only update if setPublic is explicitly set
		if setPublic != nil && fs.Public != *setPublic {
			fs.Public = *setPublic
			if err := atx.Update(ctx, &fs); err != nil {
				return fs, err
			}
		}
	}
	return fs, nil
}
