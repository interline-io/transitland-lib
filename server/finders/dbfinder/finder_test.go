package dbfinder

import (
	"context"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/tldb/querylogger"
	sq "github.com/irees/squirrel"
	"github.com/stretchr/testify/assert"
)

func TestFinder_FindFeedVersionServiceWindow(t *testing.T) {
	ctx := context.Background()
	db := testutil.MustOpenTestDB(t)
	dbf := NewFinder(&querylogger.QueryLogger{Ext: db})
	testFinder := dbf

	fvm := map[string]int{}
	fvs, err := testFinder.FindFeedVersions(ctx, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, fv := range fvs {
		fvm[fv.SHA1] = fv.ID
	}
	tcs := []struct {
		name  string
		fvid  int
		start string
		end   string
		best  string
	}{
		{
			"hart",
			fvm["c969427f56d3a645195dd8365cde6d7feae7e99b"],
			"2018-02-26", // calculated
			"2018-10-21",
			"2018-07-09",
		},
		{
			"bart",
			fvm["e535eb2b3b9ac3ef15d82c56575e914575e732e0"],
			"2018-05-26", // from feed info
			"2019-07-01", // from feed info
			"2018-06-04",
		},
		{
			"caltrain",
			fvm["d2813c293bcfd7a97dde599527ae6c62c98e66c6"],
			"2018-06-18", // calculated
			"2019-10-06",
			"2018-06-18",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			fvsw, err := testFinder.FindFeedVersionServiceWindow(ctx, tc.fvid)
			start, end, best := fvsw.StartDate, fvsw.EndDate, fvsw.FallbackWeek
			if err != nil {
				t.Fatal(err)
			}
			df := "2006-01-02"
			assert.Equal(t, tc.start, start.Format(df), "did not get expected window start")
			assert.Equal(t, tc.end, end.Format(df), "did not get expected window end")
			assert.Equal(t, tc.best, best.Format(df), "did not get expected best week in window")
			if end.Before(end) {
				t.Errorf("window end date %s before window start date %s", start.Format(df), end.Format(df))
			}
			if best.Before(start) {
				t.Errorf("best date %s before window start date %s", best.Format(df), start.Format(df))
			}
			if best.After(end) {
				t.Errorf("best date %s after window end date %s", best.Format(df), end.Format(df))
			}
		})
	}
}

func Test_alphanumeric(t *testing.T) {
	tcs := []struct {
		name   string
		value  string
		expect string
	}{
		{"ascii char", "a", "a"},
		{"ascii string", "abc", "abc"},
		{"ascii alphanumeric", "abc123", "abc123"},
		{"ascii space", "a b c", "a b c"},
		{"emdash remove", "a—b", "ab"},
		{"double space ok", "a  b", "a  b"},
		{"remove slash", "a/b", "ab"},
		{"remove single quote", "a'b", "ab"},
		{"remove dounle quote", "\"", ""},
		{"remove :", "a:b", "ab"},
		{"remove *", "a*b", "ab"},
		{"remove &", "a&b", "ab"},
		{"remove |", "a|b", "ab"},
		{"tab to space", "\t", " "},
		{"french", "Hôtel", "Hôtel"},
		{"chinese", "火车", "火车"},
		{"chinese with ascii", "abc 火车 123", "abc 火车 123"},
		{"japanese", "列車", "列車"},
		{"russian", "тренироваться", "тренироваться"},
		{"hebrew", "רכבת", "רכבת"},
		{"arabic", "قطار", "قطار"},
		{"arabic with ascii", "test قطار", "test قطار"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ret := alphanumeric(tc.value)
			if ret != tc.expect {
				t.Errorf("got '%s', expect '%s'", ret, tc.expect)
			}
		})
	}
}

func Test_az09(t *testing.T) {
	tcs := []struct {
		name   string
		value  string
		expect string
	}{
		{"plain", "hello", "hello"},
		{"underscore", "hello_world", "hello_world"},
		{"digits", "123", "123"},
		{"remove quotes", "a'b'\"c", "abc"},
		{"remove symbols", "a!b@c#d$e%f;g(h", "abcdefgh"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ret := az09(tc.value)
			if ret != tc.expect {
				t.Errorf("got '%s', expect '%s'", ret, tc.expect)
			}
		})
	}
}

func Test_pfJoinCheck_GlobalAdmin(t *testing.T) {
	baseQuery := sq.StatementBuilder.Select("*").From("current_feeds")

	t.Run("nil permFilter applies restrictions", func(t *testing.T) {
		// nil permFilter should still apply the public filter (fail closed)
		q := pfJoinCheck(baseQuery, nil)
		sql, _, _ := q.ToSql()
		// Should contain the public check
		assert.Contains(t, sql, "fsp.public = true")
		// Should contain empty IN clause (1=0 equivalent)
		assert.Contains(t, sql, "fsp.feed_id")
	})

	t.Run("empty permFilter applies restrictions", func(t *testing.T) {
		// Empty permFilter (no allowed feeds) should restrict to public only
		pf := &model.PermFilter{}
		q := pfJoinCheck(baseQuery, pf)
		sql, _, _ := q.ToSql()
		assert.Contains(t, sql, "fsp.public = true")
	})

	t.Run("permFilter with allowed feeds includes them", func(t *testing.T) {
		pf := &model.PermFilter{AllowedFeeds: []int{1, 2, 3}}
		q := pfJoinCheck(baseQuery, pf)
		sql, args, _ := q.ToSql()
		assert.Contains(t, sql, "fsp.public = true")
		assert.Contains(t, sql, "fsp.feed_id = ANY")
		// Check that feed IDs are in args
		found := false
		for _, arg := range args {
			if ids, ok := arg.([]int); ok && len(ids) == 3 {
				found = true
			}
		}
		assert.True(t, found, "expected feed IDs in query args")
	})

	t.Run("global admin bypasses permission filter", func(t *testing.T) {
		// IsGlobalAdmin should bypass the permission filter entirely
		pf := &model.PermFilter{IsGlobalAdmin: true}
		q := pfJoinCheck(baseQuery, pf)
		sql, _, _ := q.ToSql()
		// Should still join feed_states (for deleted_at check)
		assert.Contains(t, sql, "feed_states fsp")
		// Should NOT contain the public/permission OR clause
		assert.False(t, strings.Contains(sql, "fsp.public = true"), "global admin should not have public filter")
	})
}

func Test_pfJoinCheckFv_GlobalAdmin(t *testing.T) {
	baseQuery := sq.StatementBuilder.Select("*").From("feed_versions").Join("current_feeds on current_feeds.id = feed_versions.feed_id")

	t.Run("global admin bypasses permission filter", func(t *testing.T) {
		pf := &model.PermFilter{IsGlobalAdmin: true}
		q := pfJoinCheckFv(baseQuery, pf)
		sql, _, _ := q.ToSql()
		// Should still join feed_states
		assert.Contains(t, sql, "feed_states fsp")
		// Should NOT contain the public/permission OR clause
		assert.False(t, strings.Contains(sql, "fsp.public = true"), "global admin should not have public filter")
	})

	t.Run("non-admin applies restrictions", func(t *testing.T) {
		pf := &model.PermFilter{AllowedFeeds: []int{1}, AllowedFeedVersions: []int{10}}
		q := pfJoinCheckFv(baseQuery, pf)
		sql, _, _ := q.ToSql()
		assert.Contains(t, sql, "fsp.public = true")
		assert.Contains(t, sql, "feed_versions.feed_id = ANY")
		assert.Contains(t, sql, "feed_versions.id = ANY")
	})
}
