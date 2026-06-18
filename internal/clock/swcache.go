package clock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/interline-io/transitland-lib/server/caches/tzcache"
	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/tt"
	sq "github.com/irees/squirrel"
	"github.com/jmoiron/sqlx"
)

type ServiceWindow struct {
	StartDate    time.Time
	EndDate      time.Time
	FallbackWeek time.Time
	Location     *time.Location
}

type ServiceWindowCache struct {
	db          sqlx.Ext
	lock        sync.Mutex
	fvslWindows map[int]*ServiceWindow
	tzCache     *tzcache.Cache[int]
}

func NewServiceWindowCache(db sqlx.Ext) *ServiceWindowCache {
	return &ServiceWindowCache{
		db:          db,
		fvslWindows: map[int]*ServiceWindow{},
		tzCache:     tzcache.NewCache[int](),
	}
}

func (f *ServiceWindowCache) Get(ctx context.Context, fvid int) (*ServiceWindow, bool, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	a, ok := f.fvslWindows[fvid]
	if ok {
		return a, ok, nil
	}

	// Get timezone from FVSW data
	fvData, err := f.queryFv(ctx, fvid)
	if err != nil {
		return a, false, err
	}
	if fvData.Location == nil {
		return a, false, fmt.Errorf("unable to get cached default timezone for feed version %d", fvid)
	}

	a = &ServiceWindow{}
	a.Location = fvData.Location

	// Get fallback week from FVSL data
	fvslData, err := f.queryFvsl(ctx, fvid)
	if err != nil {
		return a, false, err
	}
	a.FallbackWeek = tzTruncate(fvslData.FallbackWeek, a.Location)

	// Use calculated date window if not available from FVSW
	if fvData.StartDate.IsZero() || fvData.EndDate.IsZero() {
		// Use feed info date ranges if available
		a.StartDate = tzTruncate(fvslData.StartDate, a.Location)
		a.EndDate = tzTruncate(fvslData.EndDate, a.Location)
	} else {
		// Fallback to calculated date range based on FVSL data
		a.StartDate = tzTruncate(fvData.StartDate, a.Location)
		a.EndDate = tzTruncate(fvData.EndDate, a.Location)
	}

	// Add to cache
	f.fvslWindows[fvid] = a
	return a, true, err
}

// Query feed version service level records and try to find the best date.
func (f *ServiceWindowCache) queryFv(ctx context.Context, fvid int) (ServiceWindow, error) {
	ret := ServiceWindow{}
	// Query fv fetched_at and FVSW data
	type fiQuery struct {
		FetchedAt            tt.Time
		FeedStartDate        tt.Time
		FeedEndDate          tt.Time
		EarliestCalendarDate tt.Time
		LatestCalendarDate   tt.Time
		FallbackWeek         tt.Time
		DefaultTimezone      tt.String
	}
	fvq := sq.StatementBuilder.
		Select("fv.fetched_at", "fvsw.feed_start_date", "fvsw.feed_end_date", "fvsw.earliest_calendar_date", "fvsw.latest_calendar_date", "fvsw.fallback_week", "fvsw.default_timezone").
		From("feed_versions fv").
		LeftJoin("feed_version_service_windows fvsw on fvsw.feed_version_id = fv.id").
		Where(sq.Eq{"fvsw.feed_version_id": fvid}).
		Limit(1)
	var fis []fiQuery
	if err := dbutil.Select(ctx, f.db, fvq, &fis); err != nil {
		return ret, err
	}
	if len(fis) == 0 {
		return ret, nil
	}
	fiData := fis[0]
	if fiData.FeedStartDate.Valid && fiData.FeedEndDate.Valid {
		ret.StartDate = fiData.FeedStartDate.Val
		ret.EndDate = fiData.FeedEndDate.Val
	}
	// else {
	// 	fmt.Println("using calendar start/end")
	// 	ret.StartDate = fiData.EarliestCalendarDate.Val
	// 	ret.EndDate = fiData.LatestCalendarDate.Val
	// }
	ret.Location, _ = f.tzCache.Location(fiData.DefaultTimezone.Val)
	return ret, nil
}

func (f *ServiceWindowCache) queryFvsl(ctx context.Context, fvid int) (ServiceWindow, error) {
	ret := ServiceWindow{}
	minServiceRatio := 0.75
	startDate := time.Time{}
	endDate := time.Time{}

	// Get FVSLs
	type fvslEnt struct {
		FetchedAt    tt.Time
		StartDate    tt.Time
		EndDate      tt.Time
		TotalService tt.Int
		Monday       tt.Int
		Tuesday      tt.Int
		Wednesday    tt.Int
		Thursday     tt.Int
		Friday       tt.Int
		Saturday     tt.Int
		Sunday       tt.Int
	}
	fvslQuery := sq.StatementBuilder.
		Select("fv.fetched_at", "fvsl.start_date", "fvsl.end_date",
			"monday + tuesday + wednesday + thursday + friday + saturday + sunday as total_service",
			"fvsl.monday", "fvsl.tuesday", "fvsl.wednesday", "fvsl.thursday",
			"fvsl.friday", "fvsl.saturday", "fvsl.sunday").
		From("feed_versions fv").
		Join("feed_version_service_levels fvsl on fvsl.feed_version_id = fv.id").
		Where(sq.Eq{"route_id": nil}).
		Where(sq.Eq{"fv.id": fvid}).
		OrderBy("fvsl.start_date").
		Limit(1000)
	var fvslEnts []fvslEnt
	if err := dbutil.Select(ctx, f.db, fvslQuery, &fvslEnts); err != nil {
		return ret, err
	}
	if len(fvslEnts) == 0 {
		return ret, nil
	}

	// Check if we have feed infos, otherwise calculate based on fetched week or highest service week
	// Get the week which includes fetched_at date, and the highest service week
	highestIdx := 0
	highestService := -1
	fetchedWeek := -1
	fetchedAt := fvslEnts[0].FetchedAt.Val
	for i, ent := range fvslEnts {
		sd := ent.StartDate.Val
		ed := ent.EndDate.Val
		if (sd.Before(fetchedAt) || sd.Equal(fetchedAt)) && (ed.After(fetchedAt) || ed.Equal(fetchedAt)) {
			fetchedWeek = i
		}
		if ent.TotalService.Int() > highestService {
			highestIdx = i
			highestService = ent.TotalService.Int()
		}
	}
	if fetchedWeek < 0 {
		// fmt.Println("fetched week not in fvsls, using highest week:", highestIdx, highestService)
		fetchedWeek = highestIdx
	} else {
		// fmt.Println("using fetched week:", fetchedWeek)
	}
	// If the fetched week has bad service, use highest week
	if float64(fvslEnts[fetchedWeek].TotalService.Val)/float64(highestService) < minServiceRatio {
		// fmt.Println("fetched week has poor service ratio, falling back to highest week:", fetchedWeek)
		fetchedWeek = highestIdx
	}

	// Expand window in both directions from chosen week
	startDate = fvslEnts[fetchedWeek].StartDate.Val
	endDate = fvslEnts[fetchedWeek].EndDate.Val
	for i := fetchedWeek; i < len(fvslEnts); i++ {
		ent := fvslEnts[i]
		if float64(ent.TotalService.Val)/float64(highestService) < minServiceRatio {
			break
		}
		if ent.StartDate.Val.Before(startDate) {
			startDate = ent.StartDate.Val
		}
		endDate = ent.EndDate.Val
	}
	for i := fetchedWeek - 1; i > 0; i-- {
		ent := fvslEnts[i]
		if float64(ent.TotalService.Val)/float64(highestService) < minServiceRatio {
			break
		}
		if ent.EndDate.Val.After(endDate) {
			endDate = ent.EndDate.Val
		}
		startDate = ent.StartDate.Val
	}

	// Check again to find the highest service week in the window
	// This will be used as the typical week for dates outside the window
	// bestWeek must start with a Monday
	bestWeek := fvslEnts[0].StartDate.Val
	bestService := fvslEnts[0].TotalService.Val
	hasFullServiceWeek := false

	// Helper function to check if a week falls within the service window
	isWeekInWindow := func(ent fvslEnt) bool {
		sd := ent.StartDate.Val
		ed := ent.EndDate.Val
		return (sd.Before(endDate) || sd.Equal(endDate)) && (ed.After(startDate) || ed.Equal(startDate))
	}

	// First pass: look for weeks with service on all days
	for _, ent := range fvslEnts {
		if isWeekInWindow(ent) {
			// Check if this week has service on all days
			hasFullService := ent.Monday.Val > 0 && ent.Tuesday.Val > 0 && ent.Wednesday.Val > 0 &&
				ent.Thursday.Val > 0 && ent.Friday.Val > 0 && ent.Saturday.Val > 0 && ent.Sunday.Val > 0

			if hasFullService && ent.TotalService.Val > bestService {
				bestService = ent.TotalService.Val
				bestWeek = ent.StartDate.Val
				hasFullServiceWeek = true
			}
		}
	}

	// Second pass: if no full-service week found, fall back to any week with highest service
	if !hasFullServiceWeek {
		for _, ent := range fvslEnts {
			if isWeekInWindow(ent) {
				if ent.TotalService.Val > bestService {
					bestService = ent.TotalService.Val
					bestWeek = ent.StartDate.Val
				}
			}
		}
	}
	return ServiceWindow{
		StartDate:    startDate,
		EndDate:      endDate,
		FallbackWeek: bestWeek,
	}, nil
}
