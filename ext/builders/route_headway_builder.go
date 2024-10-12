package builders

import (
	"sort"
	"time"

	"github.com/interline-io/transitland-lib/copier"
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

type riTrip struct {
	RouteID   string
	ServiceID string
	Direction uint8
}

type riKey struct {
	StopID    string
	ServiceID string
	Direction uint8
}

type RouteHeadwayBuilder struct {
	tripDetails     map[string]riTrip
	routeDepartures map[string]map[riKey][]int
	serviceDays     map[string][]string
	tripService     map[string]string
}

func NewRouteHeadwayBuilder() *RouteHeadwayBuilder {
	return &RouteHeadwayBuilder{
		tripDetails:     map[string]riTrip{},
		routeDepartures: map[string]map[riKey][]int{},
		tripService:     map[string]string{},
		serviceDays:     map[string][]string{},
	}
}

func (pp *RouteHeadwayBuilder) AfterWrite(eid string, ent tt.Entity, emap *tt.EntityMap) error {
	// Keep track of all services and departures
	switch v := ent.(type) {
	case *service.Service:
		// Use only the first 30 days of service
		startDate := v.StartDate
		for i := 0; i < 31; i++ {
			if v.IsActive(startDate) {
				d := startDate.Format("2006-01-02")
				pp.serviceDays[d] = append(pp.serviceDays[d], eid)
			}
			startDate = startDate.AddDate(0, 0, 1)
		}
	case *gtfs.Route:
		pp.routeDepartures[eid] = map[riKey][]int{}
	case *gtfs.Trip:
		// Process StopTimes assuming they will all be written
		// otherwise this breaks on journey pattern deduplication.
		for _, st := range v.StopTimes {
			stopId, ok := emap.Get("stops.txt", st.StopID.Val)
			if !ok {
				continue
			}
			rkey := riKey{
				ServiceID: v.ServiceID,
				Direction: uint8(v.DirectionID),
				StopID:    stopId,
			}
			if rd, ok := pp.routeDepartures[v.RouteID]; ok && st.DepartureTime.Valid {
				rd[rkey] = append(rd[rkey], st.DepartureTime.Int())
			}
		}
	}
	return nil
}

func (pp *RouteHeadwayBuilder) Copy(copier *copier.Copier) error {
	for rid, routeDepartures := range pp.routeDepartures {
		// log.Traceln("\n============", rid)
		// Both directions will use the same day
		departuresByService := map[string]int{}
		for k, v := range routeDepartures {
			departuresByService[k.ServiceID] += len(v)
		}
		tripsByDay := map[string]int{}
		for day, serviceids := range pp.serviceDays {
			for _, sid := range serviceids {
				tripsByDay[day] += departuresByService[sid]
			}
		}
		// Stable sort
		tripsByDaySorted := sortMap(tripsByDay)
		// log.Traceln("tripsByDay:")
		// for _, day := range tripsByDaySorted {
		// 	log.Traceln("\tday:", day, "count:", tripsByDay[day])
		// }
		// Get the highest trip count for each dow category
		dowCatDay := map[int]string{}
		dowCatCounts := map[int]int{}
		for _, day := range tripsByDaySorted {
			// parse day again to get weekday
			d, _ := time.Parse("2006-01-02", day)
			dow := d.Weekday()
			dowCat := 1
			if dow == time.Saturday {
				dowCat = 6
			} else if dow == time.Sunday {
				dowCat = 7
			}
			if _, ok := dowCatDay[dowCat]; !ok {
				dowCatDay[dowCat] = day
				dowCatCounts[dowCat] = tripsByDay[day]
			}
		}
		// For each direction...
		for direction := uint8(0); direction < 2; direction++ {
			// Find the stop with the most visits on the highest day in each dow category
			for dowCat, dowCatDay := range dowCatDay {
				d, _ := time.Parse("2006-01-02", dowCatDay)
				stopDepartures := map[string][]int{}
				serviceIds := pp.serviceDays[dowCatDay]
				for k, v := range routeDepartures {
					for _, sid := range serviceIds {
						if k.Direction == direction && k.ServiceID == sid {
							stopDepartures[k.StopID] = append(stopDepartures[k.StopID], v...)
						}
					}
				}
				// log.Traceln("routeDepartures:", routeDepartures)
				// log.Traceln("stopDepartures:", stopDepartures)
				stopsByVisits := sortMapSlice(stopDepartures)
				if len(stopsByVisits) == 0 {
					continue
				}
				// log.Traceln("direction:", direction, "dowCat:", dowCat, "dowCatDay:", dowCatDay)
				// for _, v := range stopsByVisits {
				// 	log.Traceln("\tstop:", v, "count:", len(stopDepartures[v]))
				// }
				mostVisitedStop := stopsByVisits[0]
				departures := stopDepartures[mostVisitedStop]
				sort.Ints(departures)
				// log.Debugf("rid:", rid, "dowCat:", dowCat, "dowCatDay:", day, "direction:", direction, "most visited stop:", mostVisitedStop, "sids:", serviceids)
				// log.Debugf("\tdepartures:", departures)
				// for _, departure := range departures {
				// 	wt := tt.NewSeconds(departure)
				// 	log.Debugf("\t", wt.String())
				// }
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
					rh.HeadwaySecs = tt.NewInt(ws.mid)
				}
				if _, err := copier.CopyEntity(&rh); err != nil {
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

func sortMapSlice(value map[string][]int) []string {
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
