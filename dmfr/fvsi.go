package dmfr

import (
	"fmt"
	"sort"
	"time"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/snabb/isoweek"
)

// RouteService .
type RouteService struct {
	ID        int
	RouteID   string
	StartDate time.Time
	EndDate   time.Time
	Monday    int
	Tuesday   int
	Wednesday int
	Thursday  int
	Friday    int
	Saturday  int
	Sunday    int
}

// NewFeedVersionServiceInfosFromReader .
func NewFeedVersionServiceInfosFromReader(reader tl.Reader) ([]RouteService, error) {
	results := []RouteService{}
	// Cache services
	services := map[string]*tl.Service{}
	for _, service := range tl.NewServicesFromReader(reader) {
		services[service.ServiceID] = service
	}
	// Calculate trip durations
	tripdurations := map[string]int{}
	for stoptimes := range reader.StopTimesByTripID() {
		start := stoptimes[0].DepartureTime
		end := stoptimes[len(stoptimes)-1].ArrivalTime
		// todo: frequencies
		tripdurations[stoptimes[0].TripID] = end - start
	}
	// Group durations by route,service
	routeservices := map[string]map[string]int{}
	for trip := range reader.Trips() {
		if _, ok := routeservices[trip.RouteID]; !ok {
			routeservices[trip.RouteID] = map[string]int{}
		}
		td := tripdurations[trip.TripID]
		if td > 0 {
			routeservices[trip.RouteID][trip.ServiceID] += td
		}
	}
	// Assign durations to week
	for route, v := range routeservices {
		smap := map[int][7]int{}
		for k, seconds := range v {
			service, ok := services[k]
			if !ok {
				continue
			}
			start, end := service.ServicePeriod()
			for start.Before(end) {
				if service.IsActive(start) {
					jd := toJulian(start)
					a := smap[jd]
					a[toWeekdayIndex(start)] += seconds
					smap[jd] = a
				}
				start = start.AddDate(0, 0, 1)
			}
		}
		// Group days by pattern
		imap := map[[7]int][]int{}
		for k, v := range smap {
			imap[v] = append(imap[v], k)
		}
		fmt.Println("route:", route)
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
			ranges = append(ranges, [2]int{v[start], v[len(v)-1] + 6})
			for _, r := range ranges {
				a := RouteService{
					RouteID:   route,
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
				results = append(results, a)
				// fmt.Println(a.StartDate.Format("2006-01-02"), a.EndDate.Format("2006-01-02"))
			}
		}
	}
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
