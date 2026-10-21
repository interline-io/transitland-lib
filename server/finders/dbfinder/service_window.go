package dbfinder

import (
	"context"
	"fmt"
	"sync"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tt"
	sq "github.com/irees/squirrel"
	"github.com/jmoiron/sqlx"
)

// serviceWindowCache holds each feed version's service window, chosen from the
// stats its fetch stored.
type serviceWindowCache struct {
	db      sqlx.ExtContext
	lock    sync.Mutex
	windows map[int]model.FeedServiceWindow
}

func newServiceWindowCache(db sqlx.ExtContext) *serviceWindowCache {
	return &serviceWindowCache{
		db:      db,
		windows: map[int]model.FeedServiceWindow{},
	}
}

func (c *serviceWindowCache) Get(ctx context.Context, fvid int) (model.FeedServiceWindow, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if w, ok := c.windows[fvid]; ok {
		return w, nil
	}
	type fvswRow struct {
		FetchedAt       tt.Time
		FeedStartDate   tt.Date
		FeedEndDate     tt.Date
		DefaultTimezone tt.String
	}
	fvq := sq.StatementBuilder.
		Select("fv.fetched_at", "fvsw.feed_start_date", "fvsw.feed_end_date", "fvsw.default_timezone").
		From("feed_versions fv").
		Join("feed_version_service_windows fvsw on fvsw.feed_version_id = fv.id").
		Where(sq.Eq{"fv.id": fvid}).
		Limit(1)
	var rows []fvswRow
	if err := dbutil.Select(ctx, c.db, fvq, &rows); err != nil {
		return model.FeedServiceWindow{}, err
	}
	if len(rows) == 0 {
		return model.FeedServiceWindow{}, fmt.Errorf("no service window for feed version %d", fvid)
	}
	row := rows[0]
	fvsw := dmfr.FeedVersionServiceWindow{
		FeedStartDate:   row.FeedStartDate,
		FeedEndDate:     row.FeedEndDate,
		DefaultTimezone: row.DefaultTimezone,
	}
	fvslQuery := sq.StatementBuilder.
		Select("start_date", "end_date", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday").
		From("feed_version_service_levels").
		Where(sq.Eq{"route_id": nil}).
		Where(sq.Eq{"feed_version_id": fvid}).
		OrderBy("start_date").
		Limit(1000)
	var levels []dmfr.FeedVersionServiceLevel
	if err := dbutil.Select(ctx, c.db, fvslQuery, &levels); err != nil {
		return model.FeedServiceWindow{}, err
	}
	w, err := model.NewFeedServiceWindow(fvsw, levels, row.FetchedAt.Val)
	if err != nil {
		return model.FeedServiceWindow{}, err
	}
	c.windows[fvid] = w
	return w, nil
}
