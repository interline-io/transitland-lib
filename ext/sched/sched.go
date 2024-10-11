package sched

import (
	"time"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tlutil"
	"github.com/interline-io/transitland-lib/tl/tt"
)

type tripInfo struct {
	FrequencyStarts []int
	ServiceID       string
	StartTime       tt.Seconds
	EndTime         tt.Seconds
}

type ScheduleChecker struct {
	tripInfo map[string]tripInfo
	services map[string]*tlutil.Service
}

func NewScheduleChecker() *ScheduleChecker {
	return &ScheduleChecker{
		tripInfo: map[string]tripInfo{},
		services: map[string]*tlutil.Service{},
	}
}

// Validate gets a stream of entities from Copier to build up the cache.
func (fi *ScheduleChecker) Validate(ent tl.Entity) []error {
	switch v := ent.(type) {
	case *tlutil.Service:
		fi.services[v.ServiceID] = v
	case *tl.Trip:
		ti := tripInfo{
			ServiceID: v.ServiceID,
		}
		if len(v.StopTimes) > 0 {
			ti.StartTime = v.StopTimes[0].DepartureTime
			ti.EndTime = v.StopTimes[len(v.StopTimes)-1].ArrivalTime
		}
		fi.tripInfo[v.TripID] = ti
	case *tl.Frequency:
		a := fi.tripInfo[v.TripID]
		for s := v.StartTime.Int(); s < v.EndTime.Int(); s += v.HeadwaySecs {
			a.FrequencyStarts = append(a.FrequencyStarts, s)
		}
		fi.tripInfo[v.TripID] = a
	}
	return nil
}

type dayOffset struct {
	day int
	sec int
}

func (fi *ScheduleChecker) ActiveTrips(now time.Time) []string {
	var ret []string
	dayOffsets := []dayOffset{
		{day: -1, sec: 86400},
		{day: 0, sec: 0},
	}
	for _, d := range dayOffsets {
		nowSvc := map[string]bool{}
		nowOffset := now.AddDate(0, 0, d.day)
		nowWt := tt.NewSeconds(nowOffset.Hour()*3600 + nowOffset.Minute()*60 + nowOffset.Second() + d.sec)
		for k, v := range fi.tripInfo {
			svc, ok := fi.services[v.ServiceID]
			if !ok {
				// log.Debug().
				// 	Str("service", v.ServiceID).
				// 	Str("trip", k).
				// 	Msg("no service, skipping")
				continue
			}
			// Cache if we have service on this day
			sched, ok := nowSvc[svc.ServiceID]
			if !ok {
				sched = svc.IsActive(nowOffset)
				nowSvc[svc.ServiceID] = sched
			}
			// Not scheduled
			if !sched {
				// log.Debug().
				// 	Str("date", now.Format("2006-02-03")).
				// 	Str("service", v.ServiceID).
				// 	Str("trip", k).
				// 	Msg("not scheduled, skipping")
				continue
			}

			// Might be scheduled
			found := false
			if len(v.FrequencyStarts) == 0 && nowWt.Int() >= v.StartTime.Int() && nowWt.Int() <= v.EndTime.Int() {
				// Check non-frequency based trips
				// log.Debug().
				// 	Str("date", now.Format("2006-02-03")).
				// 	Str("cur_time", nowWt.String()).
				// 	Str("trip_start", v.StartTime.String()).
				// 	Str("trip_end", v.EndTime.String()).
				// 	Str("service", v.ServiceID).
				// 	Str("trip", k).
				// 	Msg("outside time, skipping")
				found = true
			}

			// Check frequency based trips
			tripDuration := v.EndTime.Int() - v.StartTime.Int()
			for _, s := range v.FrequencyStarts {
				freqStart := s
				freqEnd := freqStart + tripDuration
				if nowWt.Int() >= freqStart && nowWt.Int() <= freqEnd {
					found = true
					break
				}
			}
			if !found {
				continue
			}
			ret = append(ret, k)
		}
	}
	return ret
}
