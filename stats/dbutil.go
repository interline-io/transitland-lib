package stats

import (
	"context"
	"database/sql"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/tldb"
)

// feedVersionDeleteBatchSize bounds how many rows one delete statement removes. The entity
// tables run to millions of rows per feed version, and a single statement over all of them
// holds its snapshot open for the duration, which stalls autovacuum database-wide.
const feedVersionDeleteBatchSize = 100_000

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
	for {
		n, err := atx.DeleteFeedVersionBatch(ctx, table, fvid, feedVersionDeleteBatchSize)
		if err != nil {
			return err
		}
		if n < feedVersionDeleteBatchSize {
			return nil
		}
	}
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
