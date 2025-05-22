package stats

import (
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tt"
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

func pdv(s string) time.Time {
	return pd(s).Val
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

var gapFvsls = []dmfr.FeedVersionServiceLevel{
	{
		StartDate: pd("2025-01-06"),
		EndDate:   pd("2025-01-12"),
		Monday:    1, Tuesday: 2, Wednesday: 3, Thursday: 4, Friday: 5, Saturday: 6, Sunday: 7,
	},
	{
		StartDate: pd("2025-01-20"),
		EndDate:   pd("2025-01-26"),
		Monday:    8, Tuesday: 9, Wednesday: 10, Thursday: 11, Friday: 12, Saturday: 13, Sunday: 14,
	},
}

var dbFvsls = []dmfr.FeedVersionServiceLevel{
	{
		StartDate: pd("2018-02-19"),
		EndDate:   pd("2018-02-25"),
		Monday:    0, Tuesday: 0, Wednesday: 0, Thursday: 0, Friday: 0, Saturday: 0, Sunday: 2843400,
	},
	{
		StartDate: pd("2018-02-26"),
		EndDate:   pd("2018-05-27"),
		Monday:    6129960, Tuesday: 6154260, Wednesday: 6129960, Thursday: 6154260, Friday: 6162180, Saturday: 3303120, Sunday: 2843400,
	},
	{
		StartDate: pd("2018-05-28"),
		EndDate:   pd("2018-06-03"),
		Monday:    2683800, Tuesday: 6154260, Wednesday: 6129960, Thursday: 6154260, Friday: 6162180, Saturday: 3303120, Sunday: 2843400,
	},
	{
		StartDate: pd("2018-06-04"),
		EndDate:   pd("2018-06-24"),
		Monday:    6129960, Tuesday: 6154260, Wednesday: 6129960, Thursday: 6154260, Friday: 6162180, Saturday: 3303120, Sunday: 2843400,
	},
	{
		StartDate: pd("2018-06-25"),
		EndDate:   pd("2018-07-01"),
		Monday:    6129960, Tuesday: 6154260, Wednesday: 6129960, Thursday: 6154260, Friday: 6162180, Saturday: 3303120, Sunday: 3130380,
	},
	{
		StartDate: pd("2018-07-02"),
		EndDate:   pd("2018-07-08"),
		Monday:    6328729, Tuesday: 6328729, Wednesday: 2932980, Thursday: 6328729, Friday: 6366349, Saturday: 3602760, Sunday: 3130380,
	},
	{
		StartDate: pd("2018-07-09"),
		EndDate:   pd("2018-09-02"),
		Monday:    6328729, Tuesday: 6328729, Wednesday: 6328729, Thursday: 6328729, Friday: 6366349, Saturday: 3602760, Sunday: 3130380,
	},
	{
		StartDate: pd("2018-09-03"),
		EndDate:   pd("2018-09-09"),
		Monday:    2932980, Tuesday: 6328729, Wednesday: 6328729, Thursday: 6328729, Friday: 6366349, Saturday: 3602760, Sunday: 3130380,
	},
	{
		StartDate: pd("2018-09-10"),
		EndDate:   pd("2018-10-14"),
		Monday:    6328729, Tuesday: 6328729, Wednesday: 6328729, Thursday: 6328729, Friday: 6366349, Saturday: 3602760, Sunday: 3130380,
	},
	{
		StartDate: pd("2018-10-15"),
		EndDate:   pd("2018-10-21"),
		Monday:    6328729, Tuesday: 6328729, Wednesday: 6328729, Thursday: 6328729, Friday: 6366349, Saturday: 3602760, Sunday: 0,
	},
}

func TestServiceLevelDaysMaxWindow(t *testing.T) {
	gapFvsls := []dmfr.FeedVersionServiceLevel{
		{
			StartDate: pd("2025-01-06"),
			EndDate:   pd("2025-01-12"),
			Monday:    1, Tuesday: 2, Wednesday: 3, Thursday: 4, Friday: 5, Saturday: 6, Sunday: 7,
		},
		{
			StartDate: pd("2025-01-20"),
			EndDate:   pd("2025-01-26"),
			Monday:    8, Tuesday: 9, Wednesday: 10, Thursday: 11, Friday: 12, Saturday: 13, Sunday: 14,
		},
	}
	_ = gapFvsls
	_ = dbFvsls
	tcs := []struct {
		name          string
		startDate     time.Time
		endDate       time.Time
		windowSize    int
		fvsls         []dmfr.FeedVersionServiceLevel
		expectStart   time.Time
		expectEnd     time.Time
		expectSeconds int
	}{
		{
			name:          "1 day",
			fvsls:         gapFvsls,
			startDate:     pdv("2025-01-01"),
			endDate:       pdv("2025-01-31"),
			windowSize:    1,
			expectStart:   pdv("2025-01-26"),
			expectEnd:     pdv("2025-01-26"),
			expectSeconds: 14,
		},
		{
			name:          "2 days",
			fvsls:         gapFvsls,
			startDate:     pdv("2025-01-01"),
			endDate:       pdv("2025-01-31"),
			windowSize:    2,
			expectStart:   pdv("2025-01-25"),
			expectEnd:     pdv("2025-01-26"),
			expectSeconds: 27,
		},
		{
			name:          "7 days",
			fvsls:         gapFvsls,
			startDate:     pdv("2025-01-01"),
			endDate:       pdv("2025-01-31"),
			windowSize:    7,
			expectStart:   pdv("2025-01-20"),
			expectEnd:     pdv("2025-01-26"),
			expectSeconds: 77,
		},
		{
			name:          "31 days",
			fvsls:         gapFvsls,
			startDate:     pdv("2025-01-01"),
			endDate:       pdv("2025-01-31"),
			windowSize:    31,
			expectStart:   pdv("2025-01-01"),
			expectEnd:     pdv("2025-01-31"),
			expectSeconds: 105,
		},
		{
			name:          "1st week",
			fvsls:         gapFvsls,
			startDate:     pdv("2025-01-06"),
			endDate:       pdv("2025-01-12"),
			windowSize:    7,
			expectStart:   pdv("2025-01-06"),
			expectEnd:     pdv("2025-01-12"),
			expectSeconds: 28,
		},
		{
			name:          "1st week open",
			fvsls:         gapFvsls,
			startDate:     pdv("2025-01-01"),
			endDate:       pdv("2025-01-18"),
			windowSize:    7,
			expectStart:   pdv("2025-01-06"),
			expectEnd:     pdv("2025-01-12"),
			expectSeconds: 28,
		},
		{
			name:          "out of bounds",
			fvsls:         gapFvsls,
			startDate:     pdv("2025-02-01"),
			endDate:       pdv("2025-02-28"),
			windowSize:    7,
			expectStart:   pdv("2025-02-22"),
			expectEnd:     pdv("2025-02-28"),
			expectSeconds: 0,
		},
		{
			name:          "db 1 day",
			fvsls:         dbFvsls,
			startDate:     pdv("2018-02-26"),
			endDate:       pdv("2018-02-26"),
			windowSize:    1,
			expectStart:   pdv("2018-02-26"),
			expectEnd:     pdv("2018-02-26"),
			expectSeconds: 6129960,
		},
		{
			name:          "db base week",
			fvsls:         dbFvsls,
			startDate:     pdv("2018-02-26"),
			endDate:       pdv("2018-03-04"),
			windowSize:    7,
			expectStart:   pdv("2018-02-26"),
			expectEnd:     pdv("2018-03-04"),
			expectSeconds: 36877140,
		},
		{
			name:          "db max",
			fvsls:         dbFvsls,
			startDate:     pdv("2018-01-01"),
			endDate:       pdv("2018-12-31"),
			windowSize:    7,
			expectStart:   pdv("2018-10-14"),
			expectEnd:     pdv("2018-10-20"),
			expectSeconds: 38414405,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			windowStart, windowEnd, windowSeconds := ServiceLevelDaysMaxWindow(tc.fvsls, tc.startDate, tc.endDate, tc.windowSize)
			assert.Equal(t, tc.expectSeconds, windowSeconds, "windowSeconds")
			assert.Equal(t, tc.expectStart, windowStart, "windowStart")
			assert.Equal(t, tc.expectEnd, windowEnd, "windowEnd")
			// fmt.Println("windowStart:", windowStart, "windowEnd:", windowEnd, "windowSeconds:", windowSeconds)
		})
	}
}

func TestServiceLevelDays(t *testing.T) {
	tcs := []struct {
		name          string
		startDate     time.Time
		endDate       time.Time
		expectSeconds int
		expectDays    int
		fvsls         []dmfr.FeedVersionServiceLevel
	}{
		{
			name:          "base day",
			fvsls:         dbFvsls,
			startDate:     pdv("2018-02-26"),
			endDate:       pdv("2018-02-26"),
			expectSeconds: 6129960,
			expectDays:    1,
		},
		{
			name:          "base week",
			fvsls:         dbFvsls,
			startDate:     pdv("2018-02-26"),
			endDate:       pdv("2018-03-04"),
			expectSeconds: 36877140,
			expectDays:    7,
		},
		{
			name:          "multiple weeks",
			fvsls:         dbFvsls,
			startDate:     pdv("2018-05-28"),
			endDate:       pdv("2018-10-05"),
			expectSeconds: 705503552,
			expectDays:    131,
		},
		{
			name:          "before",
			fvsls:         dbFvsls,
			startDate:     pdv("2010-01-01"),
			endDate:       pdv("2010-01-31"),
			expectSeconds: 0,
			expectDays:    31,
		},
		{
			name:          "gap",
			fvsls:         gapFvsls,
			startDate:     pdv("2025-01-01"),
			endDate:       pdv("2025-01-31"),
			expectSeconds: 105,
			expectDays:    31,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			totalSeconds := 0
			totalDays := 0
			endDate := tc.startDate
			for d, slevel := range ServiceLevelDays(tc.fvsls, tc.startDate, tc.endDate) {
				// t.Logf("%s: %d", d, slevel)
				totalSeconds += slevel
				totalDays += 1
				endDate = d
			}
			assert.Equal(t, tc.expectSeconds, totalSeconds, "total seconds")
			assert.Equal(t, tc.expectDays, totalDays, "total days")
			assert.Equal(t, tc.endDate.Format("2006-01-02"), endDate.Format("2006-01-02"), "end date")
		})
	}
}
