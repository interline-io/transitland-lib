package stats

import (
	"context"
	"database/sql"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/tldb"
)

// feedVersionDeleteBatchSize bounds how many rows one delete statement removes. The entity
// tables run to millions of rows per feed version, and a single statement over all of them
// holds its snapshot open for the duration, which stalls autovacuum database-wide. A variable
// so that tests can force a batch boundary without materializing millions of rows.
var feedVersionDeleteBatchSize = 100_000

// stopDeleteLocationTypes is the order stops must be deleted in: the reverse of the order the
// copier writes them (stations, then platforms/entrances/generic nodes, then boarding areas).
//
// gtfs_stops.parent_station references gtfs_stops, and the constraint is not deferrable, so it
// is checked at the end of every statement. Deleting the table in one statement used to hide
// this, since parents and children went at once. A bounded batch does not: one that removed a
// parent station but not its children fails, and because batches are taken in physical order
// the same batch would be retried forever. Emptying each level before the level it hangs from
// leaves no parent_station reference to break, so batches within a level are independent.
var stopDeleteLocationTypes = [][]int{
	{4},       // boarding areas, which hang from platforms
	{0, 2, 3}, // platforms, entrances, generic nodes, which hang from stations
	{1},       // stations
}

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
	if table == "gtfs_stops" {
		for _, locationTypes := range stopDeleteLocationTypes {
			if err := drain(ctx, func(limit int) (int64, error) {
				return atx.DeleteFeedVersionStopsBatch(ctx, fvid, locationTypes, limit)
			}); err != nil {
				return err
			}
		}
		return nil
	}
	return drain(ctx, func(limit int) (int64, error) {
		return atx.DeleteFeedVersionBatch(ctx, table, fvid, limit)
	})
}

// drain calls deleteBatch until it removes fewer rows than it was allowed to, which only
// happens once nothing is left to match.
func drain(ctx context.Context, deleteBatch func(limit int) (int64, error)) error {
	for {
		n, err := deleteBatch(feedVersionDeleteBatchSize)
		if err != nil {
			return err
		}
		if n < int64(feedVersionDeleteBatchSize) {
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
