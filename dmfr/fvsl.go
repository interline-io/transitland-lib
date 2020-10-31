package dmfr

import (
	"database/sql"
	"sort"
	"time"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/snabb/isoweek"
)

// FeedVersionServiceLevel .
type FeedVersionServiceLevel struct {
	ID        int
	RouteID   sql.NullString
	StartDate time.Time
	EndDate   time.Time
	Monday    int
	Tuesday   int
	Wednesday int
	Thursday  int
	Friday    int
	Saturday  int
	Sunday    int
	// Cached data
	AgencyName     string
	RouteShortName string
	RouteLongName  string
	RouteType      int
}

// NewFeedVersionServiceInfosFromReader .
func NewFeedVersionServiceInfosFromReader(reader tl.Reader) ([]FeedVersionServiceLevel, error) {
	results := []FeedVersionServiceLevel{}
	// Cache services
	services := map[string]*tl.Service{}
	for _, service := range tl.NewServicesFromReader(reader) {
		services[service.ServiceID] = service
	}
	// Cache frequencies; trip repeats
	freqs := map[string]int{}
	for freq := range reader.Frequencies() {
		freqs[freq.TripID] += freq.RepeatCount()
	}
	// Calculate trip durations
	tripdurations := map[string]int{}
	for stoptimes := range reader.StopTimesByTripID() {
		start := stoptimes[0].DepartureTime
		end := stoptimes[len(stoptimes)-1].ArrivalTime
		tripdurations[stoptimes[0].TripID] = end - start
	}
	// Group durations by route,service
	routeservices := map[string]map[string]int{}
	routeservices[""] = map[string]int{} // feed total
	for trip := range reader.Trips() {
		if _, ok := routeservices[trip.RouteID]; !ok {
			routeservices[trip.RouteID] = map[string]int{}
		}
		// Multiply out frequency based trips; they are scheduled or not scheduled together
		td := tripdurations[trip.TripID]
		if freq, ok := freqs[trip.TripID]; ok {
			td = td * freq
		}
		// Add to pattern
		if td > 0 {
			routeservices[trip.RouteID][trip.ServiceID] += td
			routeservices[""][trip.ServiceID] += td // Add to total
		}
	}
	// Assign durations to week for each route
	for route, v := range routeservices {
		// Calculate the total duration for each day of the service period
		smap := map[int][7]int{}
		for k, seconds := range v {
			service, ok := services[k]
			if !ok {
				continue
			}
			start, end := service.ServicePeriod()
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
		for k, v := range imap {
			if len(v) == 0 {
				continue
			}
			sort.Ints(v) // sort
			ranges := [][2]int{}
			start := 0
			for i := 0; i < len(v)-1; i++ {
				// fmt.Println(
				// 	"i:", i,
				// 	"start:", start,
				// 	"v[start]:", v[start],
				// 	"v[i]:", v[i],
				// 	"v[i]+7:", v[i]+7,
				// 	"v[i+1]:", v[i+1],
				// )
				if v[i]+7 != v[i+1] {
					ranges = append(ranges, [2]int{v[start], v[i] + 6})
					start = i + 1
				}
			}
			// Add patterns to result
			ranges = append(ranges, [2]int{v[start], v[len(v)-1] + 6})
			for _, r := range ranges {
				a := FeedVersionServiceLevel{
					StartDate: fromJulian(r[0]),
					EndDate:   fromJulian(r[1]),
					Monday:    k[0],
					Tuesday:   k[1],
					Wednesday: k[2],
					Thursday:  k[3],
					Friday:    k[4],
					Saturday:  k[5],
					Sunday:    k[6],
				}
				if route != "" {
					a.RouteID.String = route
					a.RouteID.Valid = true
				}
				results = append(results, a)
			}
		}
	}
	// Cache some helpful additional metadata
	// This will be useful for feeds that aren't imported.
	agencyNames := map[string]string{}
	for agency := range reader.Agencies() {
		agencyNames[agency.AgencyID] = agency.AgencyName
	}
	rmds := map[string]FeedVersionServiceLevel{}
	for route := range reader.Routes() {
		rmds[route.RouteID] = FeedVersionServiceLevel{
			AgencyName:     agencyNames[route.AgencyID],
			RouteShortName: route.RouteShortName,
			RouteLongName:  route.RouteLongName,
			RouteType:      route.RouteType,
		}
	}
	for i, result := range results {
		r, ok := rmds[result.RouteID.String]
		if !ok {
			continue
		}
		result.AgencyName = r.AgencyName
		result.RouteLongName = r.RouteLongName
		result.RouteShortName = r.RouteShortName
		result.RouteType = r.RouteType
		results[i] = result
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
