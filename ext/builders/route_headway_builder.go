package builders

import (
	"sort"
	"time"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/service"
	"github.com/interline-io/transitland-lib/tt"
)

type RouteHeadway struct {
	RouteID        string
	SelectedStopID string
	DirectionID    tt.Int
	HeadwaySecs    tt.Int
	DowCategory    tt.Int
	ServiceDate    tt.Date
	StopTripCount  tt.Int
	Departures     tt.Ints
	tt.MinEntity
	tt.FeedVersionEntity
}

func (ent *RouteHeadway) Filename() string {
	return "tl_route_headways.txt"
}

func (ent *RouteHeadway) TableName() string {
	return "tl_route_headways"
}

//////

type riKey struct {
	StopID    string
	ServiceID string
	Direction uint8
}

type RouteHeadwayBuilder struct {
	// Departure seconds are accumulated for every stop_time in the feed, so they are
	// stored as int32 (seconds-since-midnight fits easily) to halve this map's footprint
	// on large feeds; widened back to int only for the selected stop's stats.
	routeDepartures map[string]map[riKey][]int32
	// services holds one Service per service_id. The active-days materialization is
	// deferred to Copy and scoped to each route's own services there, instead of
	// expanding every service into a feed-wide date->services map up front.
	services map[string]*service.Service
}

func NewRouteHeadwayBuilder() *RouteHeadwayBuilder {
	return &RouteHeadwayBuilder{
		routeDepartures: map[string]map[riKey][]int32{},
		services:        map[string]*service.Service{},
	}
}

func (pp *RouteHeadwayBuilder) AfterWrite(eid string, ent tt.Entity, emap *tt.EntityMap) error {
	// Keep track of all services and departures
	switch v := ent.(type) {
	case *gtfs.Calendar:
		// Cache the service; its active days are materialized lazily in Copy.
		pp.services[eid] = service.NewService(*v, v.CalendarDates...)
	case *gtfs.Route:
		pp.routeDepartures[eid] = map[riKey][]int32{}
	case *gtfs.Trip:
		// Process StopTimes assuming they will all be written
		// otherwise this breaks on journey pattern deduplication.
		for _, st := range v.StopTimes {
			if !st.StopID.Valid {
				continue
			}
			stopId, ok := emap.Get("stops.txt", st.StopID.Val)
			if !ok {
				continue
			}
			rkey := riKey{
				ServiceID: v.ServiceID.Val,
				Direction: uint8(v.DirectionID.Val),
				StopID:    stopId,
			}
			if rd, ok := pp.routeDepartures[v.RouteID.Val]; ok && st.DepartureTime.Valid {
				rd[rkey] = append(rd[rkey], int32(st.DepartureTime.Int()))
			}
		}
	}
	return nil
}

func (pp *RouteHeadwayBuilder) Copy(copier adapters.EntityCopier) error {
	for rid, routeDepartures := range pp.routeDepartures {
		// Both directions will use the same day
		departuresByService := map[string]int{}
		for k, v := range routeDepartures {
			departuresByService[k.ServiceID] += len(v)
		}
		// Materialize active days for only this route's services (deferred from
		// accumulation), scoped to the route instead of scanning every feed service.
		routeServiceDays := map[string]map[string]bool{}
		for sid := range departuresByService {
			if svc, ok := pp.services[sid]; ok {
				routeServiceDays[sid] = serviceWindowDays(svc)
			}
		}
		tripsByDay := map[string]int{}
		for sid, n := range departuresByService {
			for day := range routeServiceDays[sid] {
				tripsByDay[day] += n
			}
		}
		// Stable sort
		tripsByDaySorted := sortMap(tripsByDay)
		// Get the highest trip count for each dow category
		dowCatDay := map[int]string{}
		for _, day := range tripsByDaySorted {
			// parse day again to get weekday
			d, _ := time.Parse("2006-01-02", day)
			dow := d.Weekday()
			dowCat := 1
			switch dow {
			case time.Saturday:
				dowCat = 6
			case time.Sunday:
				dowCat = 7
			}
			if _, ok := dowCatDay[dowCat]; !ok {
				dowCatDay[dowCat] = day
			}
		}
		// For each direction...
		for direction := uint8(0); direction < 2; direction++ {
			// Find the stop with the most visits on the highest day in each dow category
			for dowCat, day := range dowCatDay {
				d, _ := time.Parse("2006-01-02", day)
				stopDepartures := map[string][]int32{}
				for k, v := range routeDepartures {
					if k.Direction == direction && routeServiceDays[k.ServiceID][day] {
						stopDepartures[k.StopID] = append(stopDepartures[k.StopID], v...)
					}
				}
				stopsByVisits := sortMapSlice(stopDepartures)
				if len(stopsByVisits) == 0 {
					continue
				}
				mostVisitedStop := stopsByVisits[0]
				departures := intsFromInt32(stopDepartures[mostVisitedStop])
				sort.Ints(departures)
				rh := RouteHeadway{
					RouteID:        rid,
					SelectedStopID: mostVisitedStop,
					HeadwaySecs:    tt.Int{},
					DowCategory:    tt.NewInt(dowCat),
					ServiceDate:    tt.NewDate(d),
					StopTripCount:  tt.NewInt(len(departures)),
					DirectionID:    tt.NewInt(int(direction)),
					Departures:     tt.NewInts(departures),
				}
				// HeadwaySecs based on morning rush hour
				if ws, ok := getStats(departures, 21600, 36000); ok && len(departures) >= 10 {
					rh.HeadwaySecs.SetInt(ws.mid)
				}
				if err := copier.CopyEntity(&rh); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

////////

type windowStat struct {
	min int
	max int
	mid int
}

func getStats(v []int, lowerBoundInc int, upperBound int) (windowStat, bool) {
	var window []int
	for i := 0; i < len(v)-1; i++ {
		a, b := v[i], v[i+1]
		if a >= lowerBoundInc {
			window = append(window, b-a)
		}
		if b > upperBound {
			break
		}
	}
	sort.Ints(window)
	ws := windowStat{}
	if len(window) < 3 {
		return ws, false
	}
	ws.min = window[0]
	ws.max = window[len(window)-1]
	ws.mid = int(median(window))
	return ws, true
}

// must be sorted
func median(v []int) float64 {
	m := len(v) / 2
	if len(v)%2 == 0 {
		return float64(v[m])
	}
	return float64(v[m-1]+v[m]) / 2
}

// intsFromInt32 widens a departures slice (stored as int32 to bound memory across the
// whole feed) back to []int for the stats helpers and tt.Ints output.
func intsFromInt32(v []int32) []int {
	out := make([]int, len(v))
	for i, x := range v {
		out[i] = int(x)
	}
	return out
}

// serviceWindowDays returns the YYYY-MM-DD dates a service is active on within the
// first 31 days of its calendar — the window the headway day-selection uses.
func serviceWindowDays(svc *service.Service) map[string]bool {
	days := map[string]bool{}
	d := svc.StartDate.Val
	for i := 0; i < 31; i++ {
		if svc.IsActive(d) {
			days[d.Format("2006-01-02")] = true
		}
		d = d.AddDate(0, 0, 1)
	}
	return days
}

func sortMapSlice(value map[string][]int32) []string {
	type kv struct {
		Key   string
		Value int
	}
	var ss []kv
	for k, v := range value {
		ss = append(ss, kv{k, len(v)})
	}
	sort.Slice(ss, func(i, j int) bool {
		a := ss[i]
		b := ss[j]
		if a.Value == b.Value {
			return a.Key < b.Key
		}
		return a.Value > b.Value
	})
	ret := []string{}
	for _, k := range ss {
		ret = append(ret, k.Key)
	}
	return ret
}
