package stats

import (
	"encoding/json"
	"math/rand"
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tl/tt"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/stretchr/testify/assert"
)

type msi = map[string]int

func pd(s string) tt.Date {
	if s == "" {
		return tt.Date{}
	}
	a, err := tt.ParseDate(s)
	if err != nil {
		panic(err)
	}
	return a
}

func TestNewFeedVersionServiceLevelsFromReader(t *testing.T) {
	tcs := []struct {
		name         string
		url          string
		expectCounts msi
		expectResult []string
	}{
		{
			"example",
			testutil.ExampleZip.URL,
			msi{"CITY": 4, "AB": 4, "STBA": 4, "": 4},
			[]string{},
		},
		{
			"bart",
			testutil.ExampleFeedBART.URL,
			msi{"01": 12, "11": 12, "03": 12},
			[]string{
				// feed
				`{"ID":0,"StartDate":"2018-07-09","EndDate":"2018-09-02","Monday":3394620,"Tuesday":3394620,"Wednesday":3394620,"Thursday":3394620,"Friday":3394620,"Saturday":2147760,"Sunday":1567680}`,
				// a regular week
				`{"ID":0,"StartDate":"2018-11-26","EndDate":"2018-12-23","Monday":3394620,"Tuesday":3394620,"Wednesday":3394620,"Thursday":3394620,"Friday":3394620,"Saturday":2147760,"Sunday":1567680}`,
				// thanksgiving
				`{"ID":0,"StartDate":"2018-11-19","EndDate":"2018-11-25","Monday":3394620,"Tuesday":3394620,"Wednesday":3394620,"Thursday":1567680,"Friday":3394620,"Saturday":2147760,"Sunday":1567680}`,
				// end of feed
				`{"ID":0,"StartDate":"2019-07-01","EndDate":"2019-07-07","Monday":3394620,"Tuesday":0,"Wednesday":0,"Thursday":0,"Friday":0,"Saturday":0,"Sunday":0}`,
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			reader, err := tlcsv.NewReader(tc.url)
			if err != nil {
				t.Fatal(err)
			}
			results, err := NewFeedVersionServiceLevelsFromReader(reader)
			if err != nil {
				t.Error(err)
			}
			// Check for matches; uses json marshal/unmarshal for comparison and loading.
			for _, check := range tc.expectResult {
				checksl := dmfr.FeedVersionServiceLevel{}
				if err := json.Unmarshal([]byte(check), &checksl); err != nil {
					t.Error(err)
				}
				match := false
				for _, a := range results {
					if a.StartDate.String() == checksl.StartDate.String() &&
						a.EndDate.String() == checksl.EndDate.String() &&
						a.Monday == checksl.Monday &&
						a.Tuesday == checksl.Tuesday &&
						a.Wednesday == checksl.Wednesday &&
						a.Thursday == checksl.Thursday &&
						a.Friday == checksl.Friday &&
						a.Saturday == checksl.Saturday &&
						a.Sunday == checksl.Sunday {
						match = true
					}
				}
				if !match {
					t.Errorf("no match for %#v\n", check)
				}
			}
		})
	}
}

func TestServiceLevelDefaultWeek(t *testing.T) {

	fvsls := []dmfr.FeedVersionServiceLevel{
		{StartDate: pd("2022-01-03"), EndDate: pd("2022-01-09"), Monday: 1000},
		{StartDate: pd("2022-01-10"), EndDate: pd("2022-01-16"), Monday: 2000},
		{StartDate: pd("2022-01-17"), EndDate: pd("2022-01-23"), Monday: 2000},
		{StartDate: pd("2022-01-24"), EndDate: pd("2022-01-30"), Monday: 1500},
	}
	tcs := []struct {
		start  tt.Date
		end    tt.Date
		expect tt.Date
		fvsls  []dmfr.FeedVersionServiceLevel
	}{
		{pd("2022-01-03"), pd("2022-02-01"), pd("2022-01-10"), nil}, // window covers all fvsl
		{pd("2022-01-01"), pd("2022-12-31"), pd("2022-01-10"), nil}, // window covers all fvsl 2
		{pd("2022-01-01"), pd("2022-01-05"), pd("2022-01-03"), nil}, // window begin overlap
		{pd("2022-01-26"), pd("2022-02-10"), pd("2022-01-24"), nil}, // window end overlap
		{pd("2022-02-10"), pd("2022-02-14"), pd(""), nil},           // window outside all fvsl
		{pd("2021-02-10"), pd("2021-02-14"), pd(""), nil},           // window before all fvsl
		{pd("2022-01-04"), pd("2022-01-05"), pd("2022-01-03"), nil}, // window within single fvsl -- ok
		{pd("2022-01-04"), pd("2022-01-04"), pd("2022-01-03"), nil}, // window within single fvsl -- same day
		{pd("2022-01-03"), pd("2022-01-03"), pd("2022-01-03"), nil}, // window within single fvsl -- same day as start
		{pd("2022-01-01"), pd("2022-01-02"), pd(""), nil},           // window outside fvsl -- ends day before
		{pd("2022-01-31"), pd("2022-02-01"), pd(""), nil},           // window outside fvsl -- starts day after
		{pd("2022-01-30"), pd("2022-02-01"), pd("2022-01-24"), nil}, // starts last day
		{pd("2022-01-01"), pd("2022-01-03"), pd("2022-01-03"), nil}, // ends first day
		{pd("2022-01-03"), pd("2022-01-09"), pd("2022-01-03"), nil}, //
		{pd("2022-01-03"), pd("2022-01-10"), pd("2022-01-10"), nil}, //
		{pd("2022-01-03"), pd(""), pd("2022-01-10"), nil},           // open ended end
		{pd("2022-01-01"), pd(""), pd("2022-01-10"), nil},           // open ended end 2
		{pd("2022-01-24"), pd(""), pd("2022-01-24"), nil},           // open ended end 3
		{pd("2022-02-01"), pd(""), pd(""), nil},                     // open ended end - after fvsls
		{pd(""), pd(""), pd("2022-01-10"), nil},                     // no range
		{pd(""), pd("2022-02-01"), pd("2022-01-10"), nil},           // open ended start
		{pd(""), pd("2022-01-02"), pd(""), nil},                     // open ended start 2
		{pd(""), pd("2022-01-03"), pd("2022-01-03"), nil},           // open ended start 3

	}
	for _, tc := range tcs {
		t.Run("", func(t *testing.T) {
			if len(tc.fvsls) == 0 {
				tc.fvsls = fvsls[:]
			}
			// Shuffle
			rand.Shuffle(len(tc.fvsls), func(i, j int) { tc.fvsls[i], tc.fvsls[j] = tc.fvsls[j], tc.fvsls[i] })
			d, err := ServiceLevelDefaultWeek(tc.start, tc.end, tc.fvsls)
			if err != nil {
				t.Fatal(err)
			}
			assert.EqualValues(t, tc.expect.String(), d.String())
		})
	}
}
