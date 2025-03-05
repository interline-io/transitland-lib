package stats

import (
	"iter"
	"sort"
	"time"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/adapters/empty"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/service"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/snabb/isoweek"
)

func ServiceLevelDaysMaxWindow(fvsls []dmfr.FeedVersionServiceLevel, startDate time.Time, endDate time.Time, windowSize int) (time.Time, time.Time, int) {
	// Requires window size > 0
	if windowSize < 1 {
		windowSize = 1
	}
	windowSeconds := 0
	windowStart := startDate
	windowEnd := startDate

	i := 0
	window := make([]int, windowSize)
	for serviceDate, serviceLevel := range ServiceLevelDays(fvsls, startDate, endDate) {
		// Overwrite window position
		window[i%windowSize] = serviceLevel
		i++

		// Calculate window total
		tot := 0
		for _, v := range window {
			tot += v
		}

		// Update max window
		if tot >= windowSeconds {
			windowSeconds = tot
			windowEnd = serviceDate
			windowStart = serviceDate.AddDate(0, 0, -windowSize+1)
		}
	}
	return windowStart, windowEnd, windowSeconds
}

func ServiceLevelDays(fvsls []dmfr.FeedVersionServiceLevel, startDate time.Time, endDate time.Time) iter.Seq2[time.Time, int] {
	return func(yield func(time.Time, int) bool) {
		d := startDate
		for ; d.Before(endDate) || d.Equal(endDate); d = d.AddDate(0, 0, 1) {
			// Find fvsl for this date
			// TODO: optimize by keeping last matched index
			fvsl := dmfr.FeedVersionServiceLevel{}
			for _, a := range fvsls {
				if (d.After(a.StartDate.Val) || d.Equal(a.StartDate.Val)) && (d.Before(a.EndDate.Val) || d.Equal(a.EndDate.Val)) {
					fvsl = a
					break
				}
			}
			// Get fvsl day of week service seconds
			slevel := 0
			switch d.Weekday() {
			case time.Monday:
				slevel = fvsl.Monday
			case time.Tuesday:
				slevel = fvsl.Tuesday
			case time.Wednesday:
				slevel = fvsl.Wednesday
			case time.Thursday:
				slevel = fvsl.Thursday
			case time.Friday:
				slevel = fvsl.Friday
			case time.Saturday:
				slevel = fvsl.Saturday
			case time.Sunday:
				slevel = fvsl.Sunday
			}
			if !yield(d, slevel) {
				return
			}
		}
	}
}

// NewFeedVersionServiceLevelsFromReader .
func NewFeedVersionServiceLevelsFromReader(reader adapters.Reader) ([]dmfr.FeedVersionServiceLevel, error) {
	bld := NewFeedVersionServiceLevelBuilder()
	if err := copier.QuietCopy(reader, &empty.Writer{}, func(o *copier.Options) { o.AddExtension(bld) }); err != nil {
		return nil, err
	}
	results, err := bld.ServiceLevels()
	if err != nil {
		return nil, err
	}
	return results, nil
}

func ServiceLevelDefaultWeek(start tt.Date, end tt.Date, fvsls []dmfr.FeedVersionServiceLevel) (tt.Date, error) {
	// Defaults
	if start.IsZero() {
		start = tt.NewDate(time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC))
	}
	if end.IsZero() {
		end = tt.NewDate(time.Date(9999, 0, 0, 0, 0, 0, 0, time.UTC))
	}
	d := tt.Date{}
	// Get FVSLs in window
	var fvsort []dmfr.FeedVersionServiceLevel
	if start.IsZero() || end.Before(start) {
		fvsort = fvsls[:]
	} else {
		for _, fvsl := range fvsls {
			if fvsl.EndDate.Before(start) {
				continue
			}
			if fvsl.StartDate.After(end) {
				continue
			}
			fvsort = append(fvsort, fvsl)
		}
	}
	if len(fvsort) == 0 {
		return d, nil
	}
	sort.Slice(fvsort, func(i, j int) bool {
		a := fvsort[i].Total()
		b := fvsort[j].Total()
		if a == b {
			return fvsort[i].StartDate.Before(fvsort[j].StartDate)
		}
		return a > b
	})
	return fvsort[0].StartDate, nil
}

func fromJulian(day int) time.Time {
	y, m, d := isoweek.JulianToDate(day)
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func toJulian(t time.Time) int {
	yr, wk := t.ISOWeek()
	y, m, d := isoweek.StartDate(yr, wk)
	return isoweek.DateToJulian(y, m, d)
}

// return ISO Weekday - 1
func toWeekdayIndex(t time.Time) int {
	return isoweek.ISOWeekday(t.Year(), t.Month(), t.Day()) - 1
}

////////////////

type fvslTripInfo struct {
	ServiceID string
	Duration  int
}

type FeedVersionServiceLevelBuilder struct {
	services      map[string]*service.Service
	freqs         map[string]int
	tripdurations map[string]fvslTripInfo
}

func NewFeedVersionServiceLevelBuilder() *FeedVersionServiceLevelBuilder {
	return &FeedVersionServiceLevelBuilder{
		services:      map[string]*service.Service{},
		freqs:         map[string]int{},
		tripdurations: map[string]fvslTripInfo{},
	}
}

func (pp *FeedVersionServiceLevelBuilder) AfterWrite(eid string, ent tt.Entity, emap *tt.EntityMap) error {
	switch v := ent.(type) {
	case *gtfs.Calendar:
		pp.services[v.ServiceID.Val] = service.NewService(*v)
	case *gtfs.CalendarDate:
		svc, ok := pp.services[v.ServiceID.Val]
		if !ok {
			svc = &service.Service{}
			svc.Calendar = gtfs.Calendar{}
			svc.ServiceID.Set(v.ServiceID.Val)
			pp.services[v.ServiceID.Val] = svc
		}
		svc.AddCalendarDate(*v)
	case *gtfs.Frequency:
		pp.freqs[v.TripID.Val] += v.RepeatCount()
	case *gtfs.Trip:
		stoptimes := v.StopTimes
		if len(stoptimes) > 1 {
			d := stoptimes[len(stoptimes)-1].ArrivalTime.Int() - stoptimes[0].DepartureTime.Int()
			pp.tripdurations[v.TripID.Val] = fvslTripInfo{
				ServiceID: v.ServiceID.Val,
				Duration:  d,
			}
		}
	}
	return nil
}

func (pp *FeedVersionServiceLevelBuilder) Copy(adapters.EntityCopier) error {
	return nil
}

func (pp *FeedVersionServiceLevelBuilder) ServiceLevels() ([]dmfr.FeedVersionServiceLevel, error) {
	services := pp.services
	serviceTotals := map[string]int{}

	for tripId, ti := range pp.tripdurations {
		td := ti.Duration
		// Multiply out frequency based trips; they are scheduled or not scheduled together
		if freq, ok := pp.freqs[tripId]; ok {
			td = td * freq
		}
		// Add to pattern
		serviceTotals[ti.ServiceID] += td // Add to total
	}

	// Assign durations to week
	// Calculate the total duration for each day of the service period
	smap := map[int][7]int{}
	for k, seconds := range serviceTotals {
		service, ok := services[k]
		if !ok {
			continue
		}
		start, end := service.ServicePeriod()
		if start.IsZero() {
			continue
		}
		// Iterate from the first day to the last day,
		// saving the result to the Julian date index for that week
		for start.Before(end) || start.Equal(end) {
			if service.IsActive(start) {
				jd := toJulian(start)
				a := smap[jd]
				a[toWeekdayIndex(start)] += seconds
				smap[jd] = a
			}
			start = start.AddDate(0, 0, 1)
		}
	}
	// Group weeks by pattern
	imap := map[[7]int][]int{}
	for k, v := range smap {
		imap[v] = append(imap[v], k)
	}

	// Find repeating weeks
	var results []dmfr.FeedVersionServiceLevel
	for k, v := range imap {
		if len(v) == 0 {
			continue
		}
		sort.Ints(v) // sort
		// Extend the range if the next week (v[i]+7 days) is present
		// otherwise, create a new range.
		ranges := [][2]int{}
		start := 0
		for i := 0; i < len(v)-1; i++ {
			if v[i]+7 != v[i+1] {
				ranges = append(ranges, [2]int{v[start], v[i] + 6})
				start = i + 1
			}
		}
		// Add patterns to result
		ranges = append(ranges, [2]int{v[start], v[len(v)-1] + 6})
		for _, r := range ranges {
			a := dmfr.FeedVersionServiceLevel{
				StartDate: tt.NewDate(fromJulian(r[0])),
				EndDate:   tt.NewDate(fromJulian(r[1])),
				Monday:    k[0],
				Tuesday:   k[1],
				Wednesday: k[2],
				Thursday:  k[3],
				Friday:    k[4],
				Saturday:  k[5],
				Sunday:    k[6],
			}
			results = append(results, a)
		}
	}

	// Done
	return results, nil
}
