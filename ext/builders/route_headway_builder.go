package builders

import (
	"sort"
	"time"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/tl"
)

type RouteHeadway struct {
	RouteID        string
	SelectedStopID string
	DirectionID    tl.OInt
	HeadwaySecs    tl.OInt
	DowCategory    tl.OInt
	ServiceDate    tl.ODate
	StopTripCount  tl.OInt
	Departures     tl.IntSlice
	tl.MinEntity
	tl.FeedVersionEntity
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

func (pp *RouteHeadwayBuilder) AfterWrite(eid string, ent tl.Entity, emap *tl.EntityMap) error {
	// Keep track of all services and departures
	switch v := ent.(type) {
	case *tl.Service:
		// Use only the first 30 days of service
		startDate := v.StartDate
		for i := 0; i < 31; i++ {
			if v.IsActive(startDate) {
				d := startDate.Format("2006-01-02")
				pp.serviceDays[d] = append(pp.serviceDays[d], eid)
			}
			startDate = startDate.AddDate(0, 0, 1)
		}
	case *tl.Route:
		pp.routeDepartures[eid] = map[riKey][]int{}
	case *tl.Trip:
		pp.tripDetails[eid] = riTrip{
			Direction: uint8(v.DirectionID),
			ServiceID: v.ServiceID,
			RouteID:   v.RouteID,
		}
	case *tl.StopTime:
		if ti, ok := pp.tripDetails[v.TripID]; ok {
			rkey := riKey{
				ServiceID: ti.ServiceID,
				Direction: ti.Direction,
				StopID:    v.StopID,
			}
			if rd, ok := pp.routeDepartures[ti.RouteID]; ok && v.DepartureTime.Valid {
				rd[rkey] = append(rd[rkey], v.DepartureTime.Seconds)
			}
		}
	}
	return nil
}

func (pp *RouteHeadwayBuilder) Copy(copier *copier.Copier) error {
	for rid, routeDepartures := range pp.routeDepartures {
		// fmt.Println("\n============", rid)
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
		// fmt.Println("tripsByDay:")
		// for _, day := range tripsByDaySorted {
		// 	fmt.Println("\tday:", day, "count:", tripsByDay[day])
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
				// fmt.Println("routeDepartures:", routeDepartures)
				// fmt.Println("stopDepartures:", stopDepartures)
				stopsByVisits := sortMapSlice(stopDepartures)
				if len(stopsByVisits) == 0 {
					continue
				}
				// fmt.Println("direction:", direction, "dowCat:", dowCat, "dowCatDay:", dowCatDay)
				// for _, v := range stopsByVisits {
				// 	fmt.Println("\tstop:", v, "count:", len(stopDepartures[v]))
				// }
				mostVisitedStop := stopsByVisits[0]
				departures := stopDepartures[mostVisitedStop]
				sort.Ints(departures)
				// log.Debug("rid:", rid, "dowCat:", dowCat, "dowCatDay:", day, "direction:", direction, "most visited stop:", mostVisitedStop, "sids:", serviceids)
				// log.Debug("\tdepartures:", departures)
				// for _, departure := range departures {
				// 	wt := tl.NewWideTimeFromSeconds(departure)
				// 	log.Debug("\t", wt.String())
				// }
				rh := &RouteHeadway{
					RouteID:        rid,
					SelectedStopID: mostVisitedStop,
					HeadwaySecs:    tl.OInt{},
					DowCategory:    tl.NewOInt(dowCat),
					ServiceDate:    tl.NewODate(d),
					StopTripCount:  tl.NewOInt(len(departures)),
					DirectionID:    tl.NewOInt(int(direction)),
					Departures:     tl.IntSlice{Valid: true, Ints: departures},
				}
				// HeadwaySecs based on morning rush hour
				if ws, ok := getStats(departures, 21600, 36000); ok && len(departures) >= 10 {
					rh.HeadwaySecs = tl.NewOInt(ws.mid)
				}
				if _, err := copier.Writer.AddEntity(rh); err != nil {
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
