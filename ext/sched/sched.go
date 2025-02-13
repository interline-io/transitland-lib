package sched

import (
	"time"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/service"
	"github.com/interline-io/transitland-lib/tt"
)

type tripInfo struct {
	FrequencyStarts []int
	ServiceID       string
	StartTime       tt.Seconds
	EndTime         tt.Seconds
}

type ScheduleChecker struct {
	tripInfo map[string]tripInfo
	services map[string]*service.Service
}

func NewScheduleChecker() *ScheduleChecker {
	return &ScheduleChecker{
		tripInfo: map[string]tripInfo{},
		services: map[string]*service.Service{},
	}
}

// Validate gets a stream of entities from Copier to build up the cache.
func (fi *ScheduleChecker) Validate(ent tt.Entity) []error {
	switch v := ent.(type) {
	case *gtfs.Calendar:
		svc := service.NewService(*v, v.CalendarDates...)
		fi.services[v.ServiceID.Val] = svc
	case *gtfs.Trip:
		ti := tripInfo{
			ServiceID: v.ServiceID.Val,
		}
		if len(v.StopTimes) > 0 {
			ti.StartTime = v.StopTimes[0].DepartureTime
			ti.EndTime = v.StopTimes[len(v.StopTimes)-1].ArrivalTime
		}
		fi.tripInfo[v.TripID.Val] = ti
	case *gtfs.Frequency:
		if v.HeadwaySecs.Val > 0 {
			a := fi.tripInfo[v.TripID.Val]
			for s := v.StartTime.Int(); s < v.EndTime.Int(); s += v.HeadwaySecs.Int() {
				a.FrequencyStarts = append(a.FrequencyStarts, s)
			}
			fi.tripInfo[v.TripID.Val] = a
		}
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
				// log.For(ctx).Debug().
				// 	Str("service", v.ServiceID).
				// 	Str("trip", k).
				// 	Msg("no service, skipping")
				continue
			}
			// Cache if we have service on this day
			sched, ok := nowSvc[svc.ServiceID.Val]
			if !ok {
				sched = svc.IsActive(nowOffset)
				nowSvc[svc.ServiceID.Val] = sched
			}
			// Not scheduled
			if !sched {
				// log.For(ctx).Debug().
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
				// log.For(ctx).Debug().
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
