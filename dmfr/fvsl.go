package dmfr

import (
	"sort"
	"strconv"
	"time"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tt"
	"github.com/snabb/isoweek"
)

// FeedVersionServiceLevel .
type FeedVersionServiceLevel struct {
	ID            int
	FeedVersionID int
	StartDate     tl.Date
	EndDate       tl.Date
	Monday        int
	Tuesday       int
	Wednesday     int
	Thursday      int
	Friday        int
	Saturday      int
	Sunday        int
}

// EntityID .
func (fvi *FeedVersionServiceLevel) EntityID() string {
	return strconv.Itoa(fvi.ID)
}

// TableName .
func (FeedVersionServiceLevel) TableName() string {
	return "feed_version_service_levels"
}

func (fvsl *FeedVersionServiceLevel) Total() int {
	return fvsl.Monday + fvsl.Tuesday + fvsl.Wednesday + fvsl.Thursday + fvsl.Friday + fvsl.Saturday + fvsl.Sunday
}

// NewFeedVersionServiceLevelsFromReader .
func NewFeedVersionServiceLevelsFromReader(reader tl.Reader) ([]FeedVersionServiceLevel, error) {
	results := []FeedVersionServiceLevel{}
	// Cache services
	// log.Traceln("caching services")
	services := map[string]*tl.Service{}
	for _, service := range tl.NewServicesFromReader(reader) {
		services[service.ServiceID] = service
	}
	// Cache frequencies; trip repeats
	// log.Traceln("caching frequencies")
	freqs := map[string]int{}
	for freq := range reader.Frequencies() {
		freqs[freq.TripID] += freq.RepeatCount()
	}
	// Calculate trip durations
	// log.Traceln("calculating trip durations")
	tripdurations := map[string]int{}
	for stoptimes := range reader.StopTimesByTripID() {
		if len(stoptimes) < 2 {
			continue
		}
		d := stoptimes[len(stoptimes)-1].ArrivalTime.Seconds - stoptimes[0].DepartureTime.Seconds
		tripdurations[stoptimes[0].TripID] = d
	}
	// Group durations by route,service
	// log.Traceln("grouping durations")
	routeservices := map[string]map[string]int{}
	routeservices[""] = map[string]int{} // feed total
	serviceTotals := map[string]int{}
	for trip := range reader.Trips() {
		// Multiply out frequency based trips; they are scheduled or not scheduled together
		td := tripdurations[trip.TripID]
		if freq, ok := freqs[trip.TripID]; ok {
			// log.Traceln("\ttrip:", trip.TripID, "frequency repeat count:", freq)
			td = td * freq
		}
		// Add to pattern
		serviceTotals[trip.ServiceID] += td // Add to total
	}
	// Assign durations to week
	// log.Traceln("assigning durations to week")
	// log.Traceln("\troute_id:", route)
	// Calculate the total duration for each day of the service period
	// log.Printf("\t\tchecking service periods (%d)\n", len(v))
	smap := map[int][7]int{}
	for k, seconds := range serviceTotals {
		service, ok := services[k]
		if !ok {
			continue
		}
		start, end := service.ServicePeriod()
		if start.IsZero() {
			// log.Traceln("\t\t\tstart is zero! skipping", k)
			continue
		}
		// Iterate from the first day to the last day,
		// saving the result to the Julian date index for that week
		// log.Traceln("\t\t\tservice_id:", k, "start, end", start, end)
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
	// log.Traceln("\t\tgrouping weeks")
	imap := map[[7]int][]int{}
	for k, v := range smap {
		imap[v] = append(imap[v], k)
	}
	// Find repeating weeks
	// log.Traceln("\t\tfinding week repeats")
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
			a := FeedVersionServiceLevel{
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
			// log.Traceln(a)
			results = append(results, a)
		}
	}
	// Done
	return results, nil
}

func ServiceLevelDefaultWeek(start tt.Date, end tt.Date, fvsls []FeedVersionServiceLevel) (tt.Date, error) {
	// Defaults
	if start.IsZero() {
		start = tt.NewDate(time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC))
	}
	if end.IsZero() {
		end = tt.NewDate(time.Date(9999, 0, 0, 0, 0, 0, 0, time.UTC))
	}
	d := tt.Date{}
	// Get FVSLs in window
	var fvsort []FeedVersionServiceLevel
	if start.IsZero() || end.Before(start) {
		fvsort = fvsls[:]
	} else {
		for _, fvsl := range fvsls {
			if fvsl.EndDate.Before(start) {
				// log.Traceln("fvsl ends before window:", fvsl.StartDate.String(), fvsl.EndDate.String())
				continue
			}
			if fvsl.StartDate.After(end) {
				// log.Traceln("fvsl starts before window:", fvsl.StartDate.String(), fvsl.EndDate.String())
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
	// log.Traceln("window start:", start.String(), "end:", end.String())
	// for _, fvsl := range fvsort {
	// 	log.Traceln("start:", fvsl.StartDate.String(), "end:", fvsl.EndDate.String(), "total:", fvsl.Total())
	// }
	// log.Traceln("d:", fvsort[0].StartDate.String())
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
