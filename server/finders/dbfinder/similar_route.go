package dbfinder

import (
	"context"
	"sort"
	"time"

	"github.com/interline-io/transitland-lib/internal/routematch"
	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
	sq "github.com/irees/squirrel"
)

// SimilarRoutesByRouteIDs returns, for each route, the active routes in other
// feeds that share its Onestop ID, with their trips compared on the filter's
// dates when it gives any. See Route.similar_routes.
func (f *Finder) SimilarRoutesByRouteIDs(ctx context.Context, limit *int, where *model.SimilarRouteFilter, keys []int) ([][]*model.SimilarRoute, error) {
	ret := make([][]*model.SimilarRoute, len(keys))
	sources, errs := f.RoutesByIDs(ctx, keys)
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	var osids []string
	seen := map[string]bool{}
	for _, src := range sources {
		if src != nil && src.OnestopID != nil && *src.OnestopID != "" && !seen[*src.OnestopID] {
			seen[*src.OnestopID] = true
			osids = append(osids, *src.OnestopID)
		}
	}
	if len(osids) == 0 {
		return ret, nil
	}
	// Active routes only: the candidates are what global queries return now.
	cands, err := f.FindRoutes(ctx, nil, nil, nil, &model.RouteFilter{OnestopIds: osids})
	if err != nil {
		return nil, err
	}
	candsByOsid := map[string][]*model.Route{}
	for _, c := range cands {
		if c.OnestopID != nil {
			candsByOsid[*c.OnestopID] = append(candsByOsid[*c.OnestopID], c)
		}
	}
	otherFeedsOnly := where == nil || where.OtherFeedsOnly == nil || *where.OtherFeedsOnly

	// Pair each route with its candidates, and note the dates each needs.
	type job struct {
		src   *model.Route
		cands []*model.Route
		dates []time.Time
	}
	jobs := make([]job, len(keys))
	dateRoutes := map[time.Time]map[int]bool{}
	for i, src := range sources {
		if src == nil || src.OnestopID == nil {
			continue
		}
		var cs []*model.Route
		for _, c := range candsByOsid[*src.OnestopID] {
			if c.ID == src.ID || (otherFeedsOnly && c.FeedOnestopID == src.FeedOnestopID) {
				continue
			}
			cs = append(cs, c)
		}
		if len(cs) == 0 {
			continue
		}
		dates, err := f.similarRouteDates(ctx, src, where)
		if err != nil {
			return nil, err
		}
		jobs[i] = job{src: src, cands: cs, dates: dates}
		for _, d := range dates {
			if dateRoutes[d] == nil {
				dateRoutes[d] = map[int]bool{}
			}
			dateRoutes[d][src.ID] = true
			for _, c := range cs {
				dateRoutes[d][c.ID] = true
			}
		}
	}

	// Each involved route's runs on each date, and its stop patterns' stops.
	runs := map[int][]routematch.TripStart{}
	patternTrips := map[similarRoutePatternKey]similarRoutePatternTrip{}
	allDates := make([]time.Time, 0, len(dateRoutes))
	for d := range dateRoutes {
		allDates = append(allDates, d)
	}
	sort.Slice(allDates, func(i, j int) bool { return allDates[i].Before(allDates[j]) })
	for _, d := range allDates {
		routeSet := dateRoutes[d]
		routeIDs := make([]int, 0, len(routeSet))
		for id := range routeSet {
			routeIDs = append(routeIDs, id)
		}
		sort.Ints(routeIDs)
		dayRuns, err := f.similarRouteRuns(ctx, routeIDs, d, patternTrips)
		if err != nil {
			return nil, err
		}
		for rid, rs := range dayRuns {
			runs[rid] = append(runs[rid], rs...)
		}
	}
	patternStops, err := f.similarRoutePatternStops(ctx, patternTrips)
	if err != nil {
		return nil, err
	}

	opt := routematch.DefaultOptions()
	for i, j := range jobs {
		if j.src == nil {
			ret[i] = []*model.SimilarRoute{}
			continue
		}
		var out []*model.SimilarRoute
		for _, c := range j.cands {
			sr := &model.SimilarRoute{
				OtherRouteID:        c.ID,
				SameOnestopID:       true,
				RouteTypeBasicMatch: routeTypeBasicMatch(j.src.RouteType.Int(), c.RouteType.Int()),
			}
			if len(j.dates) > 0 {
				a, b := runs[j.src.ID], runs[c.ID]
				pairs := routematch.MatchTrips(a, b, opt)
				rel := model.RouteTripRelationship(routematch.Relate(len(a), len(b), len(pairs), opt))
				sr.Trips, sr.OtherTrips, sr.MatchedTrips, sr.Relationship = ptr(len(a)), ptr(len(b)), ptr(len(pairs)), &rel
				sr.StopCorrespondences = []*model.RouteStopCorrespondence{}
				for _, c := range routematch.StopCorrespondences(pairs, patternStops, patternStops, opt) {
					sr.StopCorrespondences = append(sr.StopCorrespondences, &model.RouteStopCorrespondence{
						StopID:       c.StopID,
						OtherStopID:  c.OtherStopID,
						MatchedTrips: c.MatchedTrips,
						Distance:     c.Distance,
					})
				}
			}
			out = append(out, sr)
		}
		// Most matched trips first, then by id, so a limit keeps the likeliest copies.
		sort.SliceStable(out, func(x, y int) bool {
			mx, my := -1, -1
			if out[x].MatchedTrips != nil {
				mx = *out[x].MatchedTrips
			}
			if out[y].MatchedTrips != nil {
				my = *out[y].MatchedTrips
			}
			if mx != my {
				return mx > my
			}
			return out[x].OtherRouteID < out[y].OtherRouteID
		})
		if n := finderCheckLimit(limit); uint64(len(out)) > n {
			out = out[:n]
		}
		if out == nil {
			out = []*model.SimilarRoute{}
		}
		ret[i] = out
	}
	return ret, nil
}

// The dates to compare a route's candidates on: the filter's service dates as
// given, or its relative date resolved in the route's feed version's timezone
// and the days after it. Neither moves into a fallback week.
func (f *Finder) similarRouteDates(ctx context.Context, src *model.Route, where *model.SimilarRouteFilter) ([]time.Time, error) {
	if where == nil {
		return nil, nil
	}
	var ret []time.Time
	if len(where.ServiceDates) > 0 {
		for _, d := range where.ServiceDates {
			if d != nil && d.Valid {
				ret = append(ret, civilDay(d.Val))
			}
		}
		return ret, nil
	}
	if where.RelativeDate == nil {
		return nil, nil
	}
	fvsw, err := f.FindFeedVersionServiceWindow(ctx, src.FeedVersionID)
	if err != nil {
		return nil, err
	}
	now := model.Now(ctx)
	if fvsw != nil {
		now = fvsw.NowLocal
	}
	start, err := model.ResolveRelativeDate(*where.RelativeDate, now)
	if err != nil {
		return nil, err
	}
	days := 1
	if where.Days != nil {
		days = *where.Days
	}
	for i := 0; i < days; i++ {
		ret = append(ret, civilDay(start.AddDate(0, 0, i)))
	}
	return ret, nil
}

// civilDay is a date's calendar day as midnight UTC, so dates from different
// sources compare and key alike.
func civilDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// routeTypeBasicMatch reports whether two route types have the same basic type,
// or are both in the rail family.
func routeTypeBasicMatch(a, b int) bool {
	ba, bb := tt.BasicRouteType(a), tt.BasicRouteType(b)
	if ba == bb {
		return true
	}
	rail := map[int]bool{0: true, 1: true, 2: true, 12: true}
	return rail[ba] && rail[bb]
}

// A route's stop pattern: stop_pattern_id is scoped to a feed version, and a
// route belongs to one.
type similarRoutePatternKey struct {
	RouteID       int
	StopPatternID int
}

// The trip a stop pattern's stops are read from: the lowest-numbered trip of
// the route following it on the first date it was seen. Its id is also the
// pattern's key in routematch.
type similarRoutePatternTrip struct {
	TripID        int
	FeedVersionID int
}

type similarRouteRunRow struct {
	TripID        int
	FeedVersionID int
	RouteID       int
	StopPatternID int
	DepartureTime tt.Int
	Lon           float64
	Lat           float64
}

type similarRouteFrequencyRow struct {
	TripID      int
	StartTime   int
	EndTime     int
	HeadwaySecs int
}

// similarRouteRuns returns each route's runs on a date: a trip's start and first
// stop, and one run per departure of a frequency-based trip. Each route's stop
// patterns are added to patterns, keeping the trip already chosen for one.
func (f *Finder) similarRouteRuns(ctx context.Context, routeIDs []int, date time.Time, patterns map[similarRoutePatternKey]similarRoutePatternTrip) (map[int][]routematch.TripStart, error) {
	q := sq.StatementBuilder.Select(
		"gtfs_trips.id AS trip_id",
		"gtfs_trips.feed_version_id",
		"gtfs_trips.route_id",
		"gtfs_trips.stop_pattern_id",
		"first_st.departure_time",
		"ST_X(first_stop.geometry::geometry) AS lon",
		"ST_Y(first_stop.geometry::geometry) AS lat",
	).
		From("gtfs_trips").
		JoinClause(`join lateral (
			select st.departure_time, st.stop_id
			from gtfs_stop_times st
			where st.feed_version_id = gtfs_trips.feed_version_id
				and st.trip_id = gtfs_trips.id
				and st.stop_id is not null
				and st.departure_time is not null
			order by st.stop_sequence
			limit 1
		) first_st on true`).
		Join("gtfs_stops first_stop ON first_stop.id = first_st.stop_id").
		Where(In("gtfs_trips.route_id", routeIDs)).
		OrderBy("gtfs_trips.id")
	q = serviceDateLateral(q, tt.NewDate(date))
	var rows []similarRouteRunRow
	if err := dbutil.Select(ctx, f.db, q, &rows); err != nil {
		return nil, queryErr(ctx, err)
	}
	tripIDs := make([]int, 0, len(rows))
	for _, r := range rows {
		tripIDs = append(tripIDs, r.TripID)
	}
	freqs := map[int][]similarRouteFrequencyRow{}
	if len(tripIDs) > 0 {
		fq := sq.StatementBuilder.Select("trip_id", "start_time", "end_time", "headway_secs").
			From("gtfs_frequencies").
			Where(In("trip_id", tripIDs)).
			OrderBy("trip_id", "start_time")
		var frows []similarRouteFrequencyRow
		if err := dbutil.Select(ctx, f.db, fq, &frows); err != nil {
			return nil, queryErr(ctx, err)
		}
		for _, fr := range frows {
			freqs[fr.TripID] = append(freqs[fr.TripID], fr)
		}
	}
	ret := map[int][]routematch.TripStart{}
	for _, r := range rows {
		pk := similarRoutePatternKey{RouteID: r.RouteID, StopPatternID: r.StopPatternID}
		rep, ok := patterns[pk]
		if !ok {
			// Rows come in trip id order, so this is the pattern's lowest trip today.
			rep = similarRoutePatternTrip{TripID: r.TripID, FeedVersionID: r.FeedVersionID}
			patterns[pk] = rep
		}
		first := tlxy.Point{Lon: r.Lon, Lat: r.Lat}
		if fs := freqs[r.TripID]; len(fs) > 0 {
			for _, fr := range fs {
				if fr.HeadwaySecs <= 0 {
					continue
				}
				for t := fr.StartTime; t < fr.EndTime; t += fr.HeadwaySecs {
					ret[r.RouteID] = append(ret[r.RouteID], routematch.TripStart{Date: date, Start: t, FirstStop: first, PatternKey: rep.TripID})
				}
			}
			continue
		}
		if !r.DepartureTime.Valid {
			continue
		}
		ret[r.RouteID] = append(ret[r.RouteID], routematch.TripStart{Date: date, Start: r.DepartureTime.Int(), FirstStop: first, PatternKey: rep.TripID})
	}
	return ret, nil
}

type similarRoutePatternStopRow struct {
	TripID       int
	StopSequence int
	StopID       int
	Lon          float64
	Lat          float64
}

// similarRoutePatternStops returns each stop pattern's stops in order, keyed by
// the trip they were read from.
func (f *Finder) similarRoutePatternStops(ctx context.Context, patterns map[similarRoutePatternKey]similarRoutePatternTrip) (map[int][]routematch.PatternStop, error) {
	ret := map[int][]routematch.PatternStop{}
	if len(patterns) == 0 {
		return ret, nil
	}
	var tripIDs, fvids []int
	seenFv := map[int]bool{}
	for _, p := range patterns {
		tripIDs = append(tripIDs, p.TripID)
		if !seenFv[p.FeedVersionID] {
			seenFv[p.FeedVersionID] = true
			fvids = append(fvids, p.FeedVersionID)
		}
	}
	sort.Ints(tripIDs)
	sort.Ints(fvids)
	q := sq.StatementBuilder.Select(
		"st.trip_id",
		"st.stop_sequence",
		"st.stop_id",
		"ST_X(s.geometry::geometry) AS lon",
		"ST_Y(s.geometry::geometry) AS lat",
	).
		From("gtfs_stop_times st").
		Join("gtfs_stops s ON s.id = st.stop_id").
		Where(In("st.feed_version_id", fvids)).
		Where(In("st.trip_id", tripIDs)).
		OrderBy("st.trip_id", "st.stop_sequence")
	var rows []similarRoutePatternStopRow
	if err := dbutil.Select(ctx, f.db, q, &rows); err != nil {
		return nil, queryErr(ctx, err)
	}
	for _, r := range rows {
		ret[r.TripID] = append(ret[r.TripID], routematch.PatternStop{StopID: r.StopID, Point: tlxy.Point{Lon: r.Lon, Lat: r.Lat}})
	}
	return ret, nil
}
