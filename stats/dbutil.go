package stats

import (
	"context"
	"database/sql"
	"time"

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

// OnestopIDsRetainedForFeedVersion reports whether a feed version's onestop_id stats
// should be written, per its feed's onestop_id_retention_period and the version's age.
// A feed with no feed_states row keeps them (retention 0).
func OnestopIDsRetainedForFeedVersion(ctx context.Context, atx tldb.Adapter, fvid int) (bool, error) {
	q := atx.Sqrl().
		Select(
			"feed_versions.fetched_at as fetched_at",
			"coalesce(feed_states.onestop_id_retention_period, 0) as retention",
		).
		From("feed_versions").
		LeftJoin("feed_states on feed_states.feed_id = feed_versions.feed_id").
		Where(sq.Eq{"feed_versions.id": fvid})
	qstr, qargs, err := q.ToSql()
	if err != nil {
		return false, err
	}
	var row struct {
		FetchedAt time.Time
		Retention int
	}
	if err := atx.Get(ctx, &row, qstr, qargs...); err != nil {
		return false, err
	}
	return OnestopIDsRetained(row.Retention, time.Since(row.FetchedAt)), nil
}

// EnsureFeedState gets or creates a feed state.
// New feed states default to public=true.
func EnsureFeedState(ctx context.Context, atx tldb.Adapter, feedId int) (dmfr.FeedState, error) {
	fs := dmfr.FeedState{FeedID: feedId}
	if err := atx.Get(ctx, &fs, `SELECT * FROM feed_states WHERE feed_id = ?`, feedId); err == sql.ErrNoRows {
		fs.Public = true // Default: new feeds are public
		// Seed exclude_from_global from the DMFR tag once, at creation; owned by feed_states after.
		feed := dmfr.Feed{}
		if ferr := atx.Get(ctx, &feed, `SELECT * FROM current_feeds WHERE id = ?`, feedId); ferr != nil {
			return fs, ferr
		}
		if v, _ := feed.Tags.Get("exclude_from_global_query"); v == "true" {
			fs.ExcludeFromGlobal = true
		}
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
