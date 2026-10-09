package cmds

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/feedstate"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// formatDateRanges renders ranges as "fvid:start..end", with an empty bound open-ended.
func formatDateRanges(ranges []feedstate.DateRange) []string {
	ret := []string{}
	for _, r := range ranges {
		ret = append(ret, fmt.Sprintf("%d:%s..%s", r.FeedVersionID, r.StartDate.String(), r.EndDate.String()))
	}
	return ret
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	fn := filepath.Join(t.TempDir(), "file.csv")
	require.NoError(t, os.WriteFile(fn, []byte(content), 0644))
	return fn
}

func TestReadDateRangesFile(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    []string
		wantErr bool
	}{
		{
			name:    "open-ended dates",
			content: "feed_version_id,start_date,end_date\n1,,2016-02-07\n2,2016-02-08,\n",
			want:    []string{"1:..2016-02-07", "2:2016-02-08.."},
		},
		{
			name:    "columns in any order",
			content: "end_date,feed_version_id,start_date\n2026-12-31,3,20260101\n",
			want:    []string{"3:2026-01-01..2026-12-31"},
		},
		{
			name:    "invalid feed_version_id",
			content: "feed_version_id,start_date,end_date\nx,,\n",
			wantErr: true,
		},
		{
			name:    "invalid date",
			content: "feed_version_id,start_date,end_date\n1,tomorrow,\n",
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := readDateRangesFile(writeTempFile(t, tc.content))
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, formatDateRanges(got))
		})
	}
}

func TestFeedStateManagerCommand_DateRanges(t *testing.T) {
	ctx := context.Background()
	atx := testdb.TempSqliteAdapter()
	feed := testdb.CreateTestFeed(atx, "feed-date-ranges")
	// An imported feed version whose calendars run from start through end.
	mkfv := func(sha1 string, fetchedAt time.Time, start, end string) int {
		fv := dmfr.FeedVersion{SHA1: sha1, File: sha1 + ".zip", FetchedAt: fetchedAt}
		fv.FeedID = feed.ID
		fvid := testdb.ShouldInsert(t, atx, &fv)
		fvi := dmfr.NewFeedVersionImport()
		fvi.FeedVersionID = fvid
		fvi.Success = true
		testdb.ShouldInsert(t, atx, fvi)
		fvsw := dmfr.FeedVersionServiceWindow{}
		fvsw.FeedVersionID = fvid
		fvsw.EarliestCalendarDate, _ = tt.ParseDate(start)
		fvsw.LatestCalendarDate, _ = tt.ParseDate(end)
		testdb.ShouldInsert(t, atx, &fvsw)
		return fvid
	}
	current := mkfv("current", time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), "2026-09-01", "2026-10-12")
	next := mkfv("next", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), "2026-10-13", "2027-01-31")
	getRanges := func(adapter tldb.Adapter) []string {
		var ranges []feedstate.DateRange
		testdb.ShouldSelect(t, adapter, &ranges, "select feed_id, feed_version_id, start_date, end_date from feed_version_date_ranges order by id")
		return formatDateRanges(ranges)
	}

	t.Run("sync", func(t *testing.T) {
		cmd := FeedStateManagerCommand{Adapter: atx, SyncDateRanges: true}
		require.NoError(t, cmd.Run(ctx))
		assert.Equal(t, []string{
			fmt.Sprintf("%d:..2026-10-12", current),
			fmt.Sprintf("%d:2026-10-13..", next),
		}, getRanges(atx))
	})

	t.Run("file replaces synced ranges", func(t *testing.T) {
		cmd := FeedStateManagerCommand{
			Adapter:           atx,
			SyncDateRanges:    true,
			SetDateRangesFile: writeTempFile(t, fmt.Sprintf("feed_version_id,start_date,end_date\n%d,,\n", current)),
		}
		require.NoError(t, cmd.Parse(nil))
		require.NoError(t, cmd.Run(ctx))
		assert.Equal(t, []string{fmt.Sprintf("%d:..", current)}, getRanges(atx))
	})
}
