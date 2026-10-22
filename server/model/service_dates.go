package model

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/tt"
)

// DepartureLookbackDays bounds how far back a wall calendar date query reaches
// for after-midnight departures. 1 covers departures up to 48:00; raising it
// covers services that run longer, at one more service date per query.
const DepartureLookbackDays = 1

// calendarServiceDates returns the GTFS service dates that can carry a departure
// falling on any of the given wall calendar dates.
//
// A departure at time t on service date S falls on calendar date
// S + floor(t/86400), so calendar date D is served by service dates D-k for k in
// 0..lookback. Deduplicated, so a run of consecutive dates costs
// len(run)+lookback service dates, not len(run)*(lookback+1).
func calendarServiceDates(dates []time.Time, lookback int) []time.Time {
	seen := map[string]bool{}
	var out []time.Time
	for _, d := range dates {
		for k := 0; k <= lookback; k++ {
			s := d.AddDate(0, 0, -k)
			key := s.Format("2006-01-02")
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	return out
}

// tzTruncate returns the date of s as noon in loc. Noon, since a DST change can
// skip local midnight but never noon.
func tzTruncate(s time.Time, loc *time.Location) *tt.Date {
	if loc == nil {
		log.Error().Msg("tzTruncate: loc is nil, set to UTC")
		loc = time.UTC
	}
	return ptr(tt.NewDate(time.Date(s.Year(), s.Month(), s.Day(), 12, 0, 0, 0, loc)))
}

// ResolveRelativeDate returns the date a relative date names, counted from
// nowLocal.
func ResolveRelativeDate(rd RelativeDate, nowLocal time.Time) (time.Time, error) {
	return tt.RelativeDate(nowLocal, strings.ReplaceAll(strings.ToLower(string(rd)), "_", "-"))
}

// Replaces a date outside the feed version's service window with the same
// weekday in its fallback week.
func mapIntoServiceWindow(s time.Time, fvsw *ServiceWindow) *tt.Date {
	loc := fvsw.Location
	// Compared as calendar dates in the feed version's timezone. A date scalar
	// parses as midnight UTC, and comparing instants misplaces the window's
	// first or last day.
	local := tzTruncate(s, loc)
	day := civilDate(local.Val)
	if !day.Before(civilDate(fvsw.StartDate)) && !day.After(civilDate(fvsw.EndDate)) {
		return local
	}
	dow := int(local.Val.Weekday()) - 1
	if dow < 0 {
		dow = 6
	}
	return tzTruncate(fvsw.FallbackWeek.AddDate(0, 0, dow), loc)
}

// civilDate returns t's calendar date in its own location, as midnight UTC, so
// dates compare as dates whatever their time of day.
func civilDate(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// ResolveServiceDate resolves a filter's date inputs into the service date to
// match, and the requested date before any relocation.
//
// In the feed version's terms, a relative date becomes concrete and
// use_service_window relocates a date outside the window into the fallback week.
// Without a window, the date is used as given.
func ResolveServiceDate(serviceDate *tt.Date, relativeDate *RelativeDate, useServiceWindow bool, fvsw *ServiceWindow) (*tt.Date, *tt.Date, error) {
	if fvsw == nil {
		return serviceDate, serviceDate, nil
	}
	if relativeDate != nil {
		s, err := ResolveRelativeDate(*relativeDate, fvsw.NowLocal)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid relative_date %q: %w", *relativeDate, err)
		}
		serviceDate = tzTruncate(s, fvsw.Location)
	}
	asked := serviceDate
	// Guarded: use_service_window with no date at all is a valid filter.
	if useServiceWindow && serviceDate != nil {
		serviceDate = mapIntoServiceWindow(serviceDate.Val, fvsw)
	}
	return serviceDate, asked, nil
}

// TripDate is a service date to match, with the requested dates it reports.
//
// The two differ under use_service_window, which relocates a date into the
// fallback week; several requested dates can land on the same day. Single marks
// a filter for one service_date, whose runs are not service_dates entries.
type TripDate struct {
	Query  time.Time
	Report []time.Time
	Single bool
}

// ResolveTripDates resolves a trip filter's dates into the service dates to
// match. Wall calendar `dates` also reach back DepartureLookbackDays, and take
// precedence over service_dates, which take precedence over service_date.
func ResolveTripDates(where *TripFilter, fvsw *ServiceWindow) ([]TripDate, error) {
	if where == nil {
		return nil, nil
	}
	useServiceWindow := nilOr(where.UseServiceWindow, false)
	serviceDate, askedDate, err := ResolveServiceDate(where.ServiceDate, where.RelativeDate, useServiceWindow, fvsw)
	if err != nil {
		return nil, err
	}
	src, lookback := where.ServiceDates, 0
	if len(where.Dates) > 0 {
		src, lookback = where.Dates, DepartureLookbackDays
	}
	var requested []time.Time
	for _, d := range src {
		if d != nil {
			requested = append(requested, d.Val)
		}
	}
	var out []TripDate
	at := map[string]int{}
	for _, report := range calendarServiceDates(requested, lookback) {
		query := report
		if useServiceWindow && fvsw != nil {
			query = mapIntoServiceWindow(report, fvsw).Val
		}
		key := query.Format("2006-01-02")
		i, ok := at[key]
		if !ok {
			i = len(out)
			at[key] = i
			out = append(out, TripDate{Query: query})
		}
		out[i].Report = append(out[i].Report, report)
	}
	if len(out) > 0 || serviceDate == nil {
		return out, nil
	}
	return []TripDate{{Query: serviceDate.Val, Report: []time.Time{askedDate.Val}, Single: true}}, nil
}

// SetTripRunDates sets the dates a trip runs on, as the dates its filter
// requested, replacing any the trip has.
//
// ran reports whether the trip's service is active on a query date. A trip
// matched on a single service_date runs on it, and that run is not a
// service_dates entry.
func SetTripRunDates(trip *Trip, dates []TripDate, ran func(query time.Time) bool) {
	trip.ServiceDates, trip.RunDates = nil, nil
	if len(dates) == 1 && dates[0].Single {
		trip.RunDates = []tt.Date{tt.NewDate(dates[0].Report[0])}
		return
	}
	var matched []time.Time
	for _, d := range dates {
		if ran(d.Query) {
			matched = append(matched, d.Report...)
		}
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].Before(matched[j]) })
	for _, m := range matched {
		d := tt.NewDate(m)
		trip.ServiceDates = append(trip.ServiceDates, &d)
		trip.RunDates = append(trip.RunDates, d)
	}
}

// ServiceDayQuery is one service day's share of a stop time filter: the filter
// to run, and the requested service date its stop times report. Report is nil
// for a filter without a date, which matches a stop's every stop time.
type ServiceDayQuery struct {
	Where  *StopTimeFilter
	Report *tt.Date
}

// ExpandStopTimeFilter splits a stop time filter into the service days it
// covers.
//
// Dates resolve against the feed version's timezone and clock, a window crossing
// midnight takes in the neighboring service day, and use_service_window
// relocates each day on its own. Without a window, dates are used as given in
// UTC, and next or relative_date is an error: neither resolves without a clock.
func ExpandStopTimeFilter(where *StopTimeFilter, fvsw *ServiceWindow) ([]ServiceDayQuery, error) {
	// Rewritten below; callers reuse the filter for each feed version.
	if where != nil {
		c := *where
		where = &c
	}

	// Pre-processing
	// Convert Start, End to StartTime, EndTime
	if where != nil {
		if where.Start != nil {
			where.StartTime = ptr(where.Start.Int())
			where.Start = nil
		}
		if where.End != nil {
			where.EndTime = ptr(where.End.Int())
			where.End = nil
		}
	}

	// Further processing of the StopTimeFilter
	if where != nil {
		if fvsw == nil && (where.Next != nil || where.RelativeDate != nil) {
			return nil, errors.New("next and relative_date need a service window")
		}
		var nowLocal time.Time
		loc := time.UTC
		if fvsw != nil {
			nowLocal, loc = fvsw.NowLocal, fvsw.Location
		}

		// Set ServiceDate to local timezone
		// ServiceDate is a strict GTFS calendar date
		if where.ServiceDate != nil {
			where.ServiceDate = tzTruncate(where.ServiceDate.Val, loc)
		}

		// Set Date to local timezone
		if where.Date != nil {
			where.Date = tzTruncate(where.Date.Val, loc)
		}

		// Convert relative date
		if where.RelativeDate != nil {
			s, err := ResolveRelativeDate(*where.RelativeDate, nowLocal)
			if err != nil {
				return nil, fmt.Errorf("invalid relative_date %q: %w", *where.RelativeDate, err)
			}
			where.Date = tzTruncate(s, loc)
		}

		// Convert where.Next into departure date and time window
		if where.Next != nil {
			if where.Date == nil {
				where.Date = tzTruncate(nowLocal, loc)
			}
			st := nowLocal.Hour()*3600 + nowLocal.Minute()*60 + nowLocal.Second()
			where.StartTime = ptr(st)
			where.EndTime = ptr(st + *where.Next)
		}

	}

	// Each service day is relocated on its own, so a neighboring day inside the
	// window keeps its own schedule.
	serviceDay := func(asked tt.Date) ServiceDayQuery {
		w := *where
		w.ServiceDate = &asked
		if nilOr(where.UseServiceWindow, false) && fvsw != nil {
			w.ServiceDate = mapIntoServiceWindow(asked.Val, fvsw)
		}
		return ServiceDayQuery{Where: &w, Report: &asked}
	}

	// Check if we are crossing date boundaires, and split into separate service date queries
	var whereGroups []ServiceDayQuery
	if where != nil && where.Date != nil {
		day := func(days int) tt.Date {
			return tt.NewDate(where.Date.Val.AddDate(0, 0, days))
		}
		dayStart := 0
		dayEnd := 24 * 60 * 60
		dayEndMax := 100 * 60 * 60
		whereStartTime := dayStart
		if where.StartTime != nil {
			whereStartTime = *where.StartTime
		}
		whereEndTime := dayEnd
		if where.EndTime != nil {
			whereEndTime = *where.EndTime
		}
		lookBehind := 6 * 3600
		// Query previous day
		if whereStartTime < lookBehind {
			prev := serviceDay(day(-1))
			prev.Where.StartTime = ptr(dayEnd + whereStartTime)
			prev.Where.EndTime = ptr(dayEndMax)
			whereGroups = append(whereGroups, prev)
		}
		// Query requested day
		requested := serviceDay(day(0))
		requested.Where.StartTime = ptr(max(dayStart, whereStartTime))
		requested.Where.EndTime = ptr(whereEndTime)
		whereGroups = append(whereGroups, requested)
		// Query next day
		if whereEndTime > dayEnd {
			next := serviceDay(day(1))
			next.Where.StartTime = ptr(dayStart)
			next.Where.EndTime = ptr(whereEndTime - dayEnd)
			whereGroups = append(whereGroups, next)
		}
	} else if where != nil && where.ServiceDate != nil {
		whereGroups = append(whereGroups, serviceDay(*where.ServiceDate))
	}

	// Default
	if len(whereGroups) == 0 {
		whereGroups = append(whereGroups, ServiceDayQuery{Where: where})
	}

	return whereGroups, nil
}

func ptr[T any](v T) *T {
	return &v
}

func nilOr[T any](v *T, def T) T {
	if v == nil {
		return def
	}
	return *v
}

// FirstForRoutes returns each route's first n stop times at each entity, across
// service days, in the order they happen. sts keeps its own order.
func FirstForRoutes(sts []*StopTime, n int) []*StopTime {
	// The service days' times overlap, so order by when each stop time happens,
	// as the resolver does.
	sts = slices.Clone(sts)
	slices.SortStableFunc(sts, func(a, b *StopTime) int {
		return cmp.Compare(
			a.ServiceDate.Val.Unix()+int64(a.DepartureTime.Int()),
			b.ServiceDate.Val.Unix()+int64(b.DepartureTime.Int()),
		)
	})
	type routeAtEntity struct {
		stop, location, locationGroup string
		route                         int64
	}
	seen := map[routeAtEntity]int{}
	var ret []*StopTime
	for _, st := range sts {
		k := routeAtEntity{st.StopID.Val, st.LocationID.Val, st.LocationGroupID.Val, st.RouteID.Val}
		if seen[k] < n {
			seen[k]++
			ret = append(ret, st)
		}
	}
	return ret
}
