package builders

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/tl"
)

type RouteHeadway struct {
	RouteID                      string
	SelectedStopID               string
	DirectionID                  tl.OInt
	HeadwaySecs                  tl.OInt
	DowCategory                  tl.OInt
	ServiceDate                  tl.ODate
	StopTripCount                tl.OInt
	Departures                   IntSlice
	HeadwaySecondsMorningCount   tl.OInt // Below for backward compat
	HeadwaySecondsMorningMin     tl.OInt
	HeadwaySecondsMorningMid     tl.OInt
	HeadwaySecondsMorningMax     tl.OInt
	HeadwaySecondsMiddayCount    tl.OInt
	HeadwaySecondsMiddayMin      tl.OInt
	HeadwaySecondsMiddayMid      tl.OInt
	HeadwaySecondsMiddayMax      tl.OInt
	HeadwaySecondsAfternoonCount tl.OInt
	HeadwaySecondsAfternoonMin   tl.OInt
	HeadwaySecondsAfternoonMid   tl.OInt
	HeadwaySecondsAfternoonMax   tl.OInt
	HeadwaySecondsNightCount     tl.OInt
	HeadwaySecondsNightMin       tl.OInt
	HeadwaySecondsNightMid       tl.OInt
	HeadwaySecondsNightMax       tl.OInt
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

type RouteHeadwayBuilder struct {
	serviceDays map[string][]string
	routeInfos0 map[string]*routeInfo
	routeInfos1 map[string]*routeInfo
}

func NewRouteHeadwayBuilder() *RouteHeadwayBuilder {
	return &RouteHeadwayBuilder{
		routeInfos0: map[string]*routeInfo{},
		routeInfos1: map[string]*routeInfo{},
		serviceDays: map[string][]string{},
	}
}

func (pp *RouteHeadwayBuilder) AfterWrite(eid string, ent tl.Entity, emap *tl.EntityMap) error {
	// Keep track of all services and departures
	switch v := ent.(type) {
	case *tl.Service:
		// Use only the first 30 days of service
		startDate := v.StartDate
		endDate := startDate.AddDate(0, 0, 30)
		for startDate.Before(endDate) {
			if v.IsActive(startDate) {
				d := startDate.Format("2006-01-02")
				pp.serviceDays[d] = append(pp.serviceDays[d], eid)
			}
			startDate = startDate.AddDate(0, 0, 1)
		}
	case *tl.Trip:
		ppri := pp.routeInfos0
		if v.DirectionID == 1 {
			ppri = pp.routeInfos1
		}
		ri, ok := ppri[v.RouteID]
		if !ok {
			ri = newRouteInfo()
			ppri[v.RouteID] = ri
		}
		ri.tripsByServiceID[v.ServiceID]++
		rist, ok := ri.stopDeparturesByServiceID[v.ServiceID]
		if !ok {
			rist = map[string][]int{}
			ri.stopDeparturesByServiceID[v.ServiceID] = rist
		}
		for _, st := range v.StopTimes {
			rist[st.StopID] = append(rist[st.StopID], st.DepartureTime)
		}
	}
	return nil
}

func (pp *RouteHeadwayBuilder) Copy(copier *copier.Copier) error {
	// Process each route
	emap := copier.EntityMap
	ppris := []map[string]*routeInfo{pp.routeInfos0, pp.routeInfos1}
	for direction, ppri := range ppris {
		for rid, ri := range ppri {
			dbid, ok := emap.Get("routes.txt", rid)
			if !ok {
				fmt.Println("no emap for route:", rid)
				continue
			}
			// Get the number of trips by day
			tripsByDay := map[string]int{}
			for day, serviceids := range pp.serviceDays {
				for _, sid := range serviceids {
					tripsByDay[day] += ri.tripsByServiceID[sid]
				}
			}
			// Get the highest trip count for each dow category
			dowCatDay := map[int]string{}
			dowCatCounts := map[int]int{}
			for day, count := range tripsByDay {
				// parse day again to get weekday
				d, _ := time.Parse("2006-01-02", day)
				dow := int(d.Weekday())
				dowCat := 1
				if dow == 0 {
					dowCat = 6
				} else if dow == 6 {
					dowCat = 7
				}
				// Use earliest day in ties
				cd := dowCatDay[dowCat]
				if count > dowCatCounts[dowCat] && (cd == "" || day < cd) {
					dowCatCounts[dowCat] = count
					dowCatDay[dowCat] = day
				}
			}
			// Find the stop with the most visit on the highest day in each dow category
			for dowCat, day := range dowCatDay {
				d, _ := time.Parse("2006-01-02", day)
				stopCounts := map[string]int{}
				serviceids := pp.serviceDays[day]
				for _, sid := range serviceids {
					for stopid, departures := range ri.stopDeparturesByServiceID[sid] {
						stopCounts[stopid] += len(departures)
					}
				}
				mostVisitedStop := ""
				mostVisitedStopCount := 0
				for stopid, count := range stopCounts {
					// Use earliest stopid in ties
					if count > mostVisitedStopCount && (mostVisitedStop == "" || stopid < mostVisitedStop) {
						mostVisitedStopCount = count
						mostVisitedStop = stopid
					}
				}
				stopdbid, ok := emap.Get("stops.txt", mostVisitedStop)
				if !ok {
					fmt.Println("no emap for stop:", mostVisitedStop)
					continue
				}
				departures := []int{}
				for _, serviceid := range serviceids {
					departures = append(departures, ri.stopDeparturesByServiceID[serviceid][mostVisitedStop]...)
				}
				sort.Ints(departures)
				// fmt.Println("\trid:", rid, "dowCat:", dowCat, "dowCatDay:", day, "direction:", direction, "most visited stop:", mostVisitedStop, "sids:", serviceids)
				// fmt.Println("departures:", departures)
				// for _, departure := range departures {
				// 	wt := tl.NewWideTimeFromSeconds(departure)
				// 	fmt.Println("\t", wt.String())
				// }
				_ = direction
				rh := &RouteHeadway{
					RouteID:        dbid,
					SelectedStopID: stopdbid,
					HeadwaySecs:    tl.OInt{},
					DowCategory:    tl.NewOInt(dowCat),
					ServiceDate:    tl.NewODate(d),
					StopTripCount:  tl.NewOInt(mostVisitedStopCount),
					DirectionID:    tl.NewOInt(direction),
					Departures:     IntSlice{Valid: true, Ints: departures},
				}
				// Calculate stats for backwards compat
				if ws, ok := getStats(getWindow(departures, 21600, 36000)); ok {
					rh.HeadwaySecs = tl.NewOInt(ws.mid) // also sets overall headway seconds
					rh.HeadwaySecondsMorningCount = tl.NewOInt(ws.count)
					rh.HeadwaySecondsMorningMin = tl.NewOInt(ws.min)
					rh.HeadwaySecondsMorningMid = tl.NewOInt(ws.mid)
					rh.HeadwaySecondsMorningMax = tl.NewOInt(ws.max)
				}
				if ws, ok := getStats(getWindow(departures, 36000, 57600)); ok {
					rh.HeadwaySecondsMiddayCount = tl.NewOInt(ws.count)
					rh.HeadwaySecondsMiddayMin = tl.NewOInt(ws.min)
					rh.HeadwaySecondsMiddayMid = tl.NewOInt(ws.mid)
					rh.HeadwaySecondsMiddayMax = tl.NewOInt(ws.max)
				}
				if ws, ok := getStats(getWindow(departures, 57600, 72000)); ok {
					rh.HeadwaySecondsAfternoonCount = tl.NewOInt(ws.count)
					rh.HeadwaySecondsAfternoonMin = tl.NewOInt(ws.min)
					rh.HeadwaySecondsAfternoonMid = tl.NewOInt(ws.mid)
					rh.HeadwaySecondsAfternoonMax = tl.NewOInt(ws.max)
				}
				night := []int{}
				for _, i := range departures {
					if i >= 72000 || i < 21600 {
						night = append(night, i)
					}
				}
				if ws, ok := getStats(night); ok {
					rh.HeadwaySecondsNightCount = tl.NewOInt(ws.count)
					rh.HeadwaySecondsNightMin = tl.NewOInt(ws.min)
					rh.HeadwaySecondsNightMid = tl.NewOInt(ws.mid)
					rh.HeadwaySecondsNightMax = tl.NewOInt(ws.max)
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
	min   int
	max   int
	mid   int
	count int
}

func getWindow(v []int, lowerBoundInc int, upperBound int) []int {
	f := []int{}
	for _, i := range v {
		if i >= lowerBoundInc && i < upperBound {
			f = append(f, i)
		}
	}
	return f
}

// must be sorted
func getStats(v []int) (windowStat, bool) {
	ws := windowStat{}
	count := len(v)
	if count < 3 {
		return ws, false
	}
	ws.min = 10000000
	ws.mid = int(math.Floor(median(v)))
	for _, i := range v {
		if i < ws.min {
			ws.min = i
		}
		if i > ws.max {
			ws.max = i
		}
	}
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

////////

type routeInfo struct {
	tripsByServiceID          map[string]int
	stopDeparturesByServiceID map[string]map[string][]int
}

func newRouteInfo() *routeInfo {
	return &routeInfo{
		tripsByServiceID:          map[string]int{},
		stopDeparturesByServiceID: map[string]map[string][]int{},
	}
}
