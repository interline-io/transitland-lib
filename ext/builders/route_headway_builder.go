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
		endDate := startDate.AddDate(0, 0, 30)
		for startDate.Before(endDate) {
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
		ti, ok := pp.tripDetails[v.TripID]
		if ok {
			rkey := riKey{
				ServiceID: ti.ServiceID,
				Direction: ti.Direction,
				StopID:    v.StopID,
			}
			if rd, ok := pp.routeDepartures[ti.RouteID]; ok {
				rd[rkey] = append(rd[rkey], v.DepartureTime)
			}
		}
	}
	return nil
}

func (pp *RouteHeadwayBuilder) Copy(copier *copier.Copier) error {
	for rid, ri := range pp.routeDepartures {
		// Both directions will use the same day
		_ = rid
		departuresByService := map[string]int{}
		for k, v := range ri {
			departuresByService[k.ServiceID] += len(v)
		}
		tripsByDay := map[string]int{}
		for day, serviceids := range pp.serviceDays {
			for _, sid := range serviceids {
				tripsByDay[day] += departuresByService[sid]
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
		// For each direction...
		for direction := uint8(0); direction < 2; direction++ {
			// Find the stop with the most visits on the highest day in each dow category
			for dowCat, day := range dowCatDay {
				d, _ := time.Parse("2006-01-02", day)
				stopDepartures := map[string][]int{}
				serviceids := pp.serviceDays[day]
				for k, v := range ri {
					for _, sid := range serviceids {
						if k.Direction == direction && k.ServiceID == sid {
							stopDepartures[k.StopID] = append(stopDepartures[k.StopID], v...)
						}
					}
				}
				mostVisitedStop := ""
				mostVisitedStopCount := 0
				for stopid, deps := range stopDepartures {
					// Use earliest stopid in ties
					count := len(deps)
					if count > mostVisitedStopCount && (mostVisitedStop == "" || stopid < mostVisitedStop) {
						mostVisitedStopCount = count
						mostVisitedStop = stopid
					}
				}
				if mostVisitedStop == "" {
					continue
				}
				departures := stopDepartures[mostVisitedStop]
				sort.Ints(departures)
				fmt.Println("rid:", rid, "dowCat:", dowCat, "dowCatDay:", day, "direction:", direction, "most visited stop:", mostVisitedStop, "sids:", serviceids)
				fmt.Println("\tdepartures:", departures)
				for _, departure := range departures {
					wt := tl.NewWideTimeFromSeconds(departure)
					fmt.Println("\t", wt.String())
				}
				rh := &RouteHeadway{
					RouteID:        rid,
					SelectedStopID: mostVisitedStop,
					HeadwaySecs:    tl.OInt{},
					DowCategory:    tl.NewOInt(dowCat),
					ServiceDate:    tl.NewODate(d),
					StopTripCount:  tl.NewOInt(mostVisitedStopCount),
					DirectionID:    tl.NewOInt(int(direction)),
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
