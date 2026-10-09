package feedstate

import (
	"context"
	"fmt"
	"testing"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDate parses a YYYY-MM-DD date; an empty string is a null date.
func testDate(s string) tt.Date {
	d, _ := tt.ParseDate(s)
	return d
}

// calendarWindow is a version whose calendars run from start through end.
func calendarWindow(fvid int, start, end string) versionWindow {
	return versionWindow{FeedID: 1, FeedVersionID: fvid, EarliestCalendarDate: testDate(start), LatestCalendarDate: testDate(end)}
}

// formatDateRanges renders ranges as "fvid:start..end", with an empty bound open-ended.
func formatDateRanges(ranges []DateRange) []string {
	ret := []string{}
	for _, r := range ranges {
		ret = append(ret, fmt.Sprintf("%d:%s..%s", r.FeedVersionID, r.StartDate.String(), r.EndDate.String()))
	}
	return ret
}

func TestComputeDateRanges(t *testing.T) {
	// Windows are listed most recently fetched first, the order computeDateRanges takes.
	tcs := []struct {
		name    string
		windows []versionWindow
		expect  []string
	}{
		{
			name:   "no versions",
			expect: []string{},
		},
		{
			name:    "one version answers every date",
			windows: []versionWindow{calendarWindow(1, "2026-09-01", "2026-12-31")},
			expect:  []string{"1:.."},
		},
		{
			name: "next version takes over the day after the previous one ends",
			windows: []versionWindow{
				calendarWindow(2, "2026-10-13", "2027-01-31"),
				calendarWindow(1, "2026-09-01", "2026-10-12"),
			},
			expect: []string{"1:..2026-10-12", "2:2026-10-13.."},
		},
		{
			name: "newer version takes the dates both cover",
			windows: []versionWindow{
				calendarWindow(2, "2026-10-13", "2027-03-31"),
				calendarWindow(1, "2026-09-01", "2026-12-31"),
			},
			expect: []string{"1:..2026-10-12", "2:2026-10-13.."},
		},
		{
			name: "older version answers again after a newer one ends",
			windows: []versionWindow{
				calendarWindow(2, "2026-03-01", "2026-04-30"),
				calendarWindow(1, "2026-01-01", "2026-12-31"),
			},
			expect: []string{"1:..2026-02-28", "2:2026-03-01..2026-04-30", "1:2026-05-01.."},
		},
		{
			name: "dates between versions stay with the earlier one",
			windows: []versionWindow{
				calendarWindow(2, "2026-06-01", "2026-08-31"),
				calendarWindow(1, "2026-01-01", "2026-03-31"),
			},
			expect: []string{"1:..2026-05-31", "2:2026-06-01.."},
		},
		{
			name: "newer version hides an older one it covers",
			windows: []versionWindow{
				calendarWindow(2, "2026-01-01", "2026-12-31"),
				calendarWindow(1, "2026-03-01", "2026-06-30"),
			},
			expect: []string{"2:.."},
		},
		{
			name: "feed_info narrows the calendar dates",
			windows: []versionWindow{
				{FeedID: 1, FeedVersionID: 2, FeedStartDate: testDate("2026-10-13"), EarliestCalendarDate: testDate("2026-09-01"), LatestCalendarDate: testDate("2027-01-31")},
				calendarWindow(1, "2026-09-01", "2026-12-31"),
			},
			expect: []string{"1:..2026-10-12", "2:2026-10-13.."},
		},
		{
			name: "versions without dates are skipped",
			windows: []versionWindow{
				{FeedID: 1, FeedVersionID: 3},
				{FeedID: 1, FeedVersionID: 2, FeedStartDate: testDate("2027-01-01"), LatestCalendarDate: testDate("2026-12-31")},
				calendarWindow(1, "2026-09-01", "2026-12-31"),
			},
			expect: []string{"1:.."},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, formatDateRanges(computeDateRanges(tc.windows)))
		})
	}
}

// insertFeedVersion adds a feed version fetched at fetchedAt, with an import record and a
// service window from start through end.
func insertFeedVersion(t *testing.T, adapter tldb.Adapter, feedID int, fetchedAt string, start, end string, success bool) int {
	t.Helper()
	ctx := context.Background()
	query, args, err := adapter.Sqrl().
		Insert("feed_versions").
		Columns("feed_id", "sha1", "fetched_at").
		Values(feedID, "sha1-"+fetchedAt, fetchedAt).
		Suffix("RETURNING id").
		ToSql()
	require.NoError(t, err)
	var fvid int
	require.NoError(t, adapter.DBX().QueryRowxContext(ctx, query, args...).Scan(&fvid))
	_, err = adapter.Sqrl().
		Insert("feed_version_gtfs_imports").
		Columns("feed_version_id", "success", "in_progress", "schedule_removed", "import_level", "interpolated_stop_time_count").
		Values(fvid, success, false, false, 0, 0).
		ExecContext(ctx)
	require.NoError(t, err)
	_, err = adapter.Sqrl().
		Insert("feed_version_service_windows").
		Columns("feed_version_id", "earliest_calendar_date", "latest_calendar_date").
		Values(fvid, testDate(start), testDate(end)).
		ExecContext(ctx)
	require.NoError(t, err)
	return fvid
}

func getDateRanges(t *testing.T, adapter tldb.Adapter) []string {
	t.Helper()
	var ranges []DateRange
	require.NoError(t, dbutil.Select(context.Background(), adapter.DBX(), adapter.Sqrl().
		Select("feed_id", "feed_version_id", "start_date", "end_date").
		From("feed_version_date_ranges").
		OrderBy("id"), &ranges))
	return formatDateRanges(ranges)
}

func TestManager_ComputeDateRanges(t *testing.T) {
	adapter, feedID, _ := setupTestDB(t, testFeedOnestopID, "")
	defer adapter.Close()
	fvid1 := insertFeedVersion(t, adapter, feedID, "2026-09-01", "2026-09-01", "2026-10-12", true)
	fvid2 := insertFeedVersion(t, adapter, feedID, "2026-10-01", "2026-10-13", "2027-01-31", true)
	// A failed import answers for no dates.
	insertFeedVersion(t, adapter, feedID, "2026-10-05", "2026-10-20", "2027-06-30", false)

	ranges, err := NewManager(adapter).ComputeDateRanges(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{
		fmt.Sprintf("%d:..2026-10-12", fvid1),
		fmt.Sprintf("%d:2026-10-13..", fvid2),
	}, formatDateRanges(ranges))
}

func TestManager_SetDateRanges(t *testing.T) {
	adapter, feedID, _ := setupTestDB(t, testFeedOnestopID, "")
	defer adapter.Close()
	ctx := context.Background()
	manager := NewManager(adapter)
	fvid1 := insertFeedVersion(t, adapter, feedID, "2026-09-01", "2026-09-01", "2026-10-12", true)
	fvid2 := insertFeedVersion(t, adapter, feedID, "2026-10-01", "2026-10-13", "2027-01-31", true)

	require.NoError(t, manager.SetDateRanges(ctx, []DateRange{
		{FeedID: feedID, FeedVersionID: fvid1, EndDate: testDate("2026-10-12")},
		{FeedID: feedID, FeedVersionID: fvid2, StartDate: testDate("2026-10-13")},
	}))
	assert.Equal(t, []string{
		fmt.Sprintf("%d:..2026-10-12", fvid1),
		fmt.Sprintf("%d:2026-10-13..", fvid2),
	}, getDateRanges(t, adapter))

	// Setting a feed's ranges again replaces them.
	require.NoError(t, manager.SetDateRanges(ctx, []DateRange{
		{FeedID: feedID, FeedVersionID: fvid2},
	}))
	assert.Equal(t, []string{fmt.Sprintf("%d:..", fvid2)}, getDateRanges(t, adapter))

	require.NoError(t, manager.RemoveDateRanges(ctx, fvid2))
	assert.Equal(t, []string{}, getDateRanges(t, adapter))
}
