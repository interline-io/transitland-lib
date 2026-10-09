package feedstate

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/tt"
	sq "github.com/irees/squirrel"
)

// DateRange assigns a feed version to answer for its feed from StartDate through
// EndDate. A null bound is open-ended.
type DateRange struct {
	FeedID        int     `db:"feed_id"`
	FeedVersionID int     `db:"feed_version_id"`
	StartDate     tt.Date `db:"start_date"`
	EndDate       tt.Date `db:"end_date"`
}

// ComputeDateRanges returns the date ranges of every feed with an imported feed
// version. Each date goes to the most recently fetched version covering the date.
func (m *Manager) ComputeDateRanges(ctx context.Context) ([]DateRange, error) {
	var windows []versionWindow
	q := m.adapter.Sqrl().
		Select(
			"feed_versions.feed_id",
			"feed_versions.id AS feed_version_id",
			"fvsw.feed_start_date",
			"fvsw.feed_end_date",
			"fvsw.earliest_calendar_date",
			"fvsw.latest_calendar_date",
		).
		From("feed_versions").
		Join("feed_version_gtfs_imports fvgi ON fvgi.feed_version_id = feed_versions.id").
		Join("feed_version_service_windows fvsw ON fvsw.feed_version_id = feed_versions.id").
		Where(sq.Eq{
			"fvgi.success":             true,
			"fvgi.in_progress":         false,
			"fvgi.schedule_removed":    false,
			"feed_versions.deleted_at": nil,
		}).
		OrderBy("feed_versions.feed_id", "feed_versions.fetched_at DESC", "feed_versions.id DESC")
	if err := dbutil.Select(ctx, m.adapter.DBX(), q, &windows); err != nil {
		return nil, fmt.Errorf("failed to query feed version service windows: %w", err)
	}
	// The windows arrive grouped by feed, most recently fetched first.
	var ret []DateRange
	start := 0
	for i := range windows {
		if i+1 == len(windows) || windows[i+1].FeedID != windows[i].FeedID {
			ret = append(ret, computeDateRanges(windows[start:i+1])...)
			start = i + 1
		}
	}
	return ret, nil
}

// SetDateRanges replaces the date ranges of each feed in ranges with its rows there.
// Other feeds keep theirs.
func (m *Manager) SetDateRanges(ctx context.Context, ranges []DateRange) error {
	byFeed := map[int][]DateRange{}
	for _, r := range ranges {
		byFeed[r.FeedID] = append(byFeed[r.FeedID], r)
	}
	for _, feedID := range slices.Sorted(maps.Keys(byFeed)) {
		if _, err := m.adapter.Sqrl().
			Delete("feed_version_date_ranges").
			Where(sq.Eq{"feed_id": feedID}).
			ExecContext(ctx); err != nil {
			return fmt.Errorf("failed to clear date ranges for feed %d: %w", feedID, err)
		}
		for _, r := range byFeed[feedID] {
			if _, err := m.adapter.Sqrl().
				Insert("feed_version_date_ranges").
				Columns("feed_id", "feed_version_id", "start_date", "end_date").
				Values(r.FeedID, r.FeedVersionID, r.StartDate, r.EndDate).
				ExecContext(ctx); err != nil {
				return fmt.Errorf("failed to set date ranges for feed %d: %w", feedID, err)
			}
		}
	}
	return nil
}

// RemoveDateRanges removes a feed version's date ranges. Its feed's active version
// answers for those dates until the ranges are next set.
func (m *Manager) RemoveDateRanges(ctx context.Context, feedVersionID int) error {
	if _, err := m.adapter.Sqrl().
		Delete("feed_version_date_ranges").
		Where(sq.Eq{"feed_version_id": feedVersionID}).
		ExecContext(ctx); err != nil {
		return fmt.Errorf("failed to remove date ranges for feed version %d: %w", feedVersionID, err)
	}
	return nil
}

// versionWindow is the service window of an imported feed version.
type versionWindow struct {
	FeedID               int     `db:"feed_id"`
	FeedVersionID        int     `db:"feed_version_id"`
	FeedStartDate        tt.Date `db:"feed_start_date"`
	FeedEndDate          tt.Date `db:"feed_end_date"`
	EarliestCalendarDate tt.Date `db:"earliest_calendar_date"`
	LatestCalendarDate   tt.Date `db:"latest_calendar_date"`
}

// covers returns the first and last dates a feed version covers, as the feed versions
// covers filter defines them: from the later of its feed_info start and earliest
// calendar date, to the earlier of its feed_info end and latest calendar date.
func (w versionWindow) covers() (time.Time, time.Time, bool) {
	var starts, ends []time.Time
	for _, d := range []tt.Date{w.FeedStartDate, w.EarliestCalendarDate} {
		if d.Valid {
			starts = append(starts, d.Val)
		}
	}
	for _, d := range []tt.Date{w.FeedEndDate, w.LatestCalendarDate} {
		if d.Valid {
			ends = append(ends, d.Val)
		}
	}
	if len(starts) == 0 || len(ends) == 0 {
		return time.Time{}, time.Time{}, false
	}
	start := slices.MaxFunc(starts, time.Time.Compare)
	end := slices.MinFunc(ends, time.Time.Compare)
	return start, end, !start.After(end)
}

// computeDateRanges gives each date to the first of one feed's windows covering the
// date, so windows must be in order of precedence. The ranges tile every date: a
// range extends across any gap after its own, and the first and last ranges are
// open-ended.
func computeDateRanges(windows []versionWindow) []DateRange {
	type span struct {
		window     versionWindow
		start, end time.Time
	}
	var spans []span
	var bounds []time.Time
	for _, w := range windows {
		start, end, ok := w.covers()
		if !ok {
			continue
		}
		spans = append(spans, span{w, start, end})
		bounds = append(bounds, start, end.AddDate(0, 0, 1))
	}
	slices.SortFunc(bounds, time.Time.Compare)
	bounds = slices.CompactFunc(bounds, time.Time.Equal)

	// Each pair of adjacent bounds is a run of dates that the same windows cover.
	var ret []DateRange
	for _, day := range bounds[:max(len(bounds)-1, 0)] {
		i := slices.IndexFunc(spans, func(s span) bool {
			return !day.Before(s.start) && !day.After(s.end)
		})
		if i < 0 {
			continue
		}
		w := spans[i].window
		if n := len(ret); n > 0 {
			if ret[n-1].FeedVersionID == w.FeedVersionID {
				continue
			}
			ret[n-1].EndDate = tt.NewDate(day.AddDate(0, 0, -1))
		}
		ret = append(ret, DateRange{FeedID: w.FeedID, FeedVersionID: w.FeedVersionID, StartDate: tt.NewDate(day)})
	}
	if len(ret) > 0 {
		ret[0].StartDate = tt.Date{}
	}
	return ret
}
