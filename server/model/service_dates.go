package model

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/tt"
)

// How a finder resolves the dates in a filter into the GTFS service dates it
// matches. Every finder shares this, so they agree on which dates a query means;
// a finder only decides which services run on a date.

// DepartureLookbackDays bounds how far back a wall calendar date query reaches
// for after-midnight departures. 1 covers departures up to 48:00; raising it
// covers services that run longer, at one more service date per query.
const DepartureLookbackDays = 1

// calendarServiceDates returns the GTFS service dates that can carry a departure
// falling on any of the given wall calendar dates. A departure at time t on
// service date S falls on calendar date S + floor(t/86400), so calendar date D
// is served by service dates D-k for k in 0..lookback. Deduplicated, so a run of
// consecutive dates costs len(run)+lookback service dates, not
// len(run)*(lookback+1).
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

// TzTruncate returns the date of s as noon in loc. Noon, since a DST change can
// skip local midnight but never noon.
func TzTruncate(s time.Time, loc *time.Location) *tt.Date {
	if loc == nil {
		log.Error().Msg("TzTruncate: loc is nil, set to UTC")
		loc = time.UTC
	}
	d := tt.NewDate(time.Date(s.Year(), s.Month(), s.Day(), 12, 0, 0, 0, loc))
	return &d
}

// ResolveRelativeDate returns the date a relative date names, counted from
// nowLocal.
func ResolveRelativeDate(rd RelativeDate, nowLocal time.Time) (time.Time, error) {
	return tt.RelativeDate(nowLocal, strings.ReplaceAll(strings.ToLower(string(rd)), "_", "-"))
}

// Replaces a date outside the feed version's service window with the same
// weekday in its fallback week.
func mapIntoServiceWindow(s time.Time, fvsw *ServiceWindow) *tt.Date {
	loc := fvsw.NowLocal.Location()
	// Compared as calendar dates in the feed version's timezone. A date scalar
	// parses as midnight UTC, and comparing instants misplaces the window's
	// first or last day.
	local := TzTruncate(s, loc)
	day := civilDate(local.Val)
	if !day.Before(civilDate(fvsw.StartDate)) && !day.After(civilDate(fvsw.EndDate)) {
		return local
	}
	dow := int(local.Val.Weekday()) - 1
	if dow < 0 {
		dow = 6
	}
	return TzTruncate(fvsw.FallbackWeek.AddDate(0, 0, dow), loc)
}

// civilDate returns t's calendar date in its own location, as midnight UTC, so
// dates compare as dates whatever their time of day.
func civilDate(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// ResolveServiceDate resolves a filter's date inputs into the one service date
// to match, in the feed version's terms: a relative date becomes concrete, and
// use_service_window relocates a date outside the window into the fallback week.
// Also returns the requested date, before any relocation. Without a window, the
// date is used as given.
func ResolveServiceDate(serviceDate *tt.Date, relativeDate *RelativeDate, useServiceWindow bool, fvsw *ServiceWindow) (*tt.Date, *tt.Date, error) {
	if fvsw == nil {
		return serviceDate, serviceDate, nil
	}
	if relativeDate != nil {
		s, err := ResolveRelativeDate(*relativeDate, fvsw.NowLocal)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid relative_date %q: %w", *relativeDate, err)
		}
		serviceDate = TzTruncate(s, fvsw.NowLocal.Location())
	}
	asked := serviceDate
	// Guarded: use_service_window with no date at all is a valid filter.
	if useServiceWindow && serviceDate != nil {
		serviceDate = mapIntoServiceWindow(serviceDate.Val, fvsw)
	}
	return serviceDate, asked, nil
}

// TripDate is a service date to match and the dates it is reported as. They
// differ under use_service_window, which relocates a date into the fallback
// week; several requested dates can land on the same day. Single marks a filter
// for one service_date, whose runs are not service_dates entries.
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
	useServiceWindow := where.UseServiceWindow != nil && *where.UseServiceWindow
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
	if len(out) > 0 {
		return out, nil
	}
	serviceDate, askedDate, err := ResolveServiceDate(where.ServiceDate, where.RelativeDate, useServiceWindow, fvsw)
	if err != nil || serviceDate == nil {
		return nil, err
	}
	return []TripDate{{Query: serviceDate.Val, Report: []time.Time{askedDate.Val}, Single: true}}, nil
}

// SetTripRunDates records the dates a trip runs on, among the dates its filter
// resolved to, as the requested dates. ran reports whether the trip's service is
// active on a query date. A single service_date is a run but not a service_dates
// entry, and the trip was matched on it, so ran is not consulted.
func SetTripRunDates(trip *Trip, dates []TripDate, ran func(query time.Time) bool) {
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
