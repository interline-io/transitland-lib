package dbfinder

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/server/caches/tzcache"
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
	tzCache *tzcache.Cache[int]
}

func newServiceWindowCache(db sqlx.ExtContext) *serviceWindowCache {
	return &serviceWindowCache{
		db:      db,
		windows: map[int]model.FeedServiceWindow{},
		tzCache: tzcache.NewCache[int](),
	}
}

func (c *serviceWindowCache) Get(ctx context.Context, fvid int) (model.FeedServiceWindow, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if w, ok := c.windows[fvid]; ok {
		return w, nil
	}
	fvq := sq.StatementBuilder.
		Select("fv.fetched_at", "fvsw.feed_start_date", "fvsw.feed_end_date", "fvsw.default_timezone").
		From("feed_versions fv").
		Join("feed_version_service_windows fvsw on fvsw.feed_version_id = fv.id").
		Where(sq.Eq{"fv.id": fvid}).
		Limit(1)
	var rows []struct {
		FetchedAt tt.Time
		dmfr.FeedVersionServiceWindow
	}
	if err := dbutil.Select(ctx, c.db, fvq, &rows); err != nil {
		return model.FeedServiceWindow{}, err
	}
	if len(rows) == 0 {
		return model.FeedServiceWindow{}, fmt.Errorf("no service window for feed version %d", fvid)
	}
	row := rows[0]
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
	w, err := model.NewFeedServiceWindow(row.FeedVersionServiceWindow, levels, row.FetchedAt.Val, c.loadLocation)
	if err != nil {
		return model.FeedServiceWindow{}, fmt.Errorf("feed version %d: %w", fvid, err)
	}
	c.windows[fvid] = w
	return w, nil
}

// loadLocation resolves a timezone through the cache, so feed versions in one
// timezone share its location.
func (c *serviceWindowCache) loadLocation(name string) (*time.Location, error) {
	loc, ok := c.tzCache.Location(name)
	if !ok {
		return nil, fmt.Errorf("unknown timezone %q", name)
	}
	return loc, nil
}
