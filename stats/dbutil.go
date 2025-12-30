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

// EnsureFeedState gets or creates a feed state.
// New feed states default to public=true.
func EnsureFeedState(ctx context.Context, atx tldb.Adapter, feedId int) (dmfr.FeedState, error) {
	fs := dmfr.FeedState{FeedID: feedId}
	if err := atx.Get(ctx, &fs, `SELECT * FROM feed_states WHERE feed_id = ?`, feedId); err == sql.ErrNoRows {
		fs.Public = true // Default: new feeds are public
		fs.ID, err = atx.Insert(ctx, &fs)
		if err != nil {
			return fs, err
		}
	} else if err != nil {
		return fs, err
	}
	return fs, nil
}

// SetFeedStatePublic sets the public flag on an existing feed state.
func SetFeedStatePublic(ctx context.Context, atx tldb.Adapter, feedId int, public bool) error {
	fs := dmfr.FeedState{}
	if err := atx.Get(ctx, &fs, `SELECT * FROM feed_states WHERE feed_id = ?`, feedId); err != nil {
		return err
	}
	if fs.Public != public {
		fs.Public = public
		return atx.Update(ctx, &fs)
	}
	return nil
}
