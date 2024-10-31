package stats

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/tldb"
)

func FeedVersionTableDelete(atx tldb.Adapter, table string, fvid int, ifExists bool) error {
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
	_, err := atx.Sqrl().Delete(table).Where(where).Exec()
	if err != nil {
		return err
	}
	return nil
}

func GetFeedState(atx tldb.Adapter, feedId int) (dmfr.FeedState, error) {
	// Get state, create if necessary
	fs := dmfr.FeedState{FeedID: feedId}
	if err := atx.Get(&fs, `SELECT * FROM feed_states WHERE feed_id = ?`, feedId); err == sql.ErrNoRows {
		fs.ID, err = atx.Insert(&fs)
		if err != nil {
			return fs, err
		}
	} else if err != nil {
		return fs, err
	}
	return fs, nil
}
