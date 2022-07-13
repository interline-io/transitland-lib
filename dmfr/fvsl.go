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

// NewFeedVersionServiceInfosFromReader .
func NewFeedVersionServiceInfosFromReader(reader tl.Reader) ([]FeedVersionServiceLevel, error) {
	results := []FeedVersionServiceLevel{}
	// Cache services
	// fmt.Println("caching services")
	services := map[string]*tl.Service{}
	for _, service := range tl.NewServicesFromReader(reader) {
		services[service.ServiceID] = service
	}
	// Cache frequencies; trip repeats
	// fmt.Println("caching frequencies")
	freqs := map[string]int{}
	for freq := range reader.Frequencies() {
		freqs[freq.TripID] += freq.RepeatCount()
	}
	// Calculate trip durations
	// fmt.Println("calculating trip durations")
	tripdurations := map[string]int{}
	for stoptimes := range reader.StopTimesByTripID() {
		if len(stoptimes) < 2 {
			continue
		}
		d := stoptimes[len(stoptimes)-1].ArrivalTime.Seconds - stoptimes[0].DepartureTime.Seconds
		tripdurations[stoptimes[0].TripID] = d
	}
	// Group durations by route,service
	// fmt.Println("grouping durations")
	routeservices := map[string]map[string]int{}
	routeservices[""] = map[string]int{} // feed total
	serviceTotals := map[string]int{}
	for trip := range reader.Trips() {
		// Multiply out frequency based trips; they are scheduled or not scheduled together
		td := tripdurations[trip.TripID]
		if freq, ok := freqs[trip.TripID]; ok {
			// fmt.Println("\ttrip:", trip.TripID, "frequency repeat count:", freq)
			td = td * freq
		}
		// Add to pattern
		serviceTotals[trip.ServiceID] += td // Add to total
	}
	// Assign durations to week
	// fmt.Println("assigning durations to week")
	// fmt.Println("\troute_id:", route)
	// Calculate the total duration for each day of the service period
	// fmt.Printf("\t\tchecking service periods (%d)\n", len(v))
	smap := map[int][7]int{}
	for k, seconds := range serviceTotals {
		service, ok := services[k]
		if !ok {
			continue
		}
		start, end := service.ServicePeriod()
		if start.IsZero() {
			// fmt.Println("\t\t\tstart is zero! skipping", k)
			continue
		}
		// Iterate from the first day to the last day,
		// saving the result to the Julian date index for that week
		// fmt.Println("\t\t\tservice_id:", k, "start, end", start, end)
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
	// fmt.Println("\t\tgrouping weeks")
	imap := map[[7]int][]int{}
	for k, v := range smap {
		imap[v] = append(imap[v], k)
	}
	// Find repeating weeks
	// fmt.Println("\t\tfinding week repeats")
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
			// fmt.Println(a)
			results = append(results, a)
		}
	}
	// Done
	return results, nil
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
