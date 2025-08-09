package dbfinder

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/tldb/querylogger"
	"github.com/interline-io/transitland-mw/testutil"
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
