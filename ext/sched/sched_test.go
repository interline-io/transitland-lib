package sched

import (
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/adapters/empty"
	"github.com/interline-io/transitland-lib/adapters/tlcsv"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func newTestScheduleSchecker(path string) (*ScheduleChecker, error) {
	ex := NewScheduleChecker()
	r, err := tlcsv.NewReader(path)
	if err != nil {
		return nil, err
	}
	cp, err := copier.NewCopier(r, &empty.Writer{}, copier.Options{})
	if err != nil {
		return nil, err
	}
	cp.AddExtension(ex)
	cpResult := cp.Copy()
	if cpResult.WriteError != nil {
		return nil, err
	}
	return ex, nil
}

func TestScheduleChecker(t *testing.T) {
	ex, err := newTestScheduleSchecker(testutil.RelPath("testdata/rt/ct.zip"))
	if err != nil {
		t.Fatal(err)
	}
	tz, _ := time.LoadLocation("America/Los_Angeles")
	type tc struct {
		name string
		when time.Time
		exp  []string
	}
	extc := []tc{
		{
			name: "midday",
			when: time.Date(2023, 11, 7, 17, 30, 0, 0, tz),
			exp:  []string{"412", "411", "410", "309", "308", "710", "311", "312", "127", "310", "125", "126", "709"},
		},
		{
			name: "saturday overnight",
			when: time.Date(2023, 11, 11, 0, 30, 0, 0, tz),
			exp:  []string{"145", "146"},
		},
		{
			name: "sunday overnight",
			when: time.Date(2023, 11, 12, 0, 30, 0, 0, tz),
			exp:  []string{"280", "284", "281"},
		},
	}
	for _, tc := range extc {
		t.Run(tc.name, func(t *testing.T) {
			stats := ex.ActiveTrips(tc.when)
			assert.ElementsMatch(t, tc.exp, stats)
		})
	}
	freqEx, err := newTestScheduleSchecker(testutil.RelPath("testdata/example.zip"))
	if err != nil {
		t.Fatal(err)
	}
	freqtc := []tc{
		// 6:05 is just after the 6:00 trips for STBA, CITY1, CITY2 started
		{
			when: time.Date(2007, 01, 10, 6, 5, 0, 0, tz),
			exp:  []string{"STBA", "CITY2", "CITY1"},
		},
		// 6:25 is after the first STBA trip ends,
		// but within the 26 minute duration of CITY1/CITY2 starting at 6:00am
		{
			when: time.Date(2007, 01, 10, 6, 25, 0, 0, tz),
			exp:  []string{"CITY2", "CITY1"},
		},
		// 6:28 is in between any scheduled trips
		{
			when: time.Date(2007, 01, 10, 6, 28, 0, 0, tz),
			exp:  []string{},
		},
		// 04:00 is before any trips start
		{
			when: time.Date(2007, 01, 10, 4, 0, 0, 0, tz),
			exp:  []string{},
		},
		// 23:00 is after all trips end
		{
			when: time.Date(2007, 01, 10, 23, 0, 0, 0, tz),
			exp:  []string{},
		},
		// 21:30 is the last scheduled trip, so 21:40 will have all three routes
		{
			when: time.Date(2007, 01, 10, 21, 40, 0, 0, tz),
			exp:  []string{"STBA", "CITY2", "CITY1"},
		},
		// 21:30 + 25 minutes, 21:55, should be after STBA ends
		{
			when: time.Date(2007, 01, 10, 21, 55, 0, 0, tz),
			exp:  []string{"CITY2", "CITY1"},
		},
		// 16:15 should have multiple trips of CITY1, CITY2 and 1 trip of STBA
		{
			when: time.Date(2007, 01, 10, 16, 15, 0, 0, tz),
			exp:  []string{"STBA", "CITY2", "CITY1"},
		},
		// Weekend
		{
			when: time.Date(2007, 01, 13, 16, 0, 0, 0, tz),
			exp:  []string{"AAMV4", "STBA", "CITY2", "CITY1"},
		},
	}
	for _, tc := range freqtc {
		t.Run("frequency "+tc.name, func(t *testing.T) {
			stats := freqEx.ActiveTrips(tc.when)
			assert.ElementsMatch(t, tc.exp, stats)
		})
	}

}
