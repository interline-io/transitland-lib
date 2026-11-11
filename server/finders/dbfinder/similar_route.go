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
	for i := range ret {
		ret[i] = []*model.SimilarRoute{}
	}
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

	// Pair each route with its candidates and the dates to compare them on.
	// Each route's trips are compared over its own dates; a candidate's runs on
	// the days either side count as partners for runs past midnight, but not
	// toward its trips.
	type job struct {
		src     *model.Route
		cands   []*model.Route
		dates   map[time.Time]bool
		partner map[time.Time]bool
	}
	jobs := make([]*job, len(keys))
	involved := map[int]bool{}
	allDays := map[time.Time]bool{}
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
		j := &job{src: src, cands: cs}
		dates := f.similarRouteDates(ctx, src, where)
		if len(dates) > 0 {
			j.dates, j.partner = map[time.Time]bool{}, map[time.Time]bool{}
			for _, d := range dates {
				j.dates[d] = true
				for _, p := range []time.Time{d.AddDate(0, 0, -1), d, d.AddDate(0, 0, 1)} {
					j.partner[p] = true
					allDays[p] = true
				}
			}
			involved[src.ID] = true
			for _, c := range cs {
				involved[c.ID] = true
			}
		}
		jobs[i] = j
	}

	runs, patternTrips, err := f.similarRouteRuns(ctx, involved, allDays)
	if err != nil {
		return nil, err
	}
	onDays := func(routeID int, days map[time.Time]bool) []routematch.TripStart {
		var out []routematch.TripStart
		for _, r := range runs[routeID] {
			if days[r.day] {
				out = append(out, r.start)
			}
		}
		return out
	}

	// Compare each route with its candidates; stop correspondences wait until
	// every pairing is known, so only patterns that matched load their stops.
	type pending struct {
		sr    *model.SimilarRoute
		pairs []routematch.Pair
	}
	opt := routematch.DefaultOptions()
	var toAlign []pending
	usedPatterns := map[int]bool{}
	for i, j := range jobs {
		if j == nil {
			continue
		}
		// The route against all of each candidate feed's rows together, since a
		// feed can split one route into rows that no single one covers.
		type feedCompare struct {
			trips, matched int
			rel            model.RouteTripRelationship
		}
		byFeed := map[string]*feedCompare{}
		if j.dates != nil {
			feedRoutes := map[string][]int{}
			for _, c := range j.cands {
				feedRoutes[c.FeedOnestopID] = append(feedRoutes[c.FeedOnestopID], c.ID)
			}
			a, aPartner := onDays(j.src.ID, j.dates), onDays(j.src.ID, j.partner)
			for feed, ids := range feedRoutes {
				var b, bPartner []routematch.TripStart
				for _, id := range ids {
					b = append(b, onDays(id, j.dates)...)
					bPartner = append(bPartner, onDays(id, j.partner)...)
				}
				matched := len(routematch.MatchTrips(a, bPartner, opt))
				otherMatched := len(routematch.MatchTrips(b, aPartner, opt))
				byFeed[feed] = &feedCompare{
					trips:   len(b),
					matched: matched,
					rel:     model.RouteTripRelationship(routematch.Relate(len(a), len(b), matched, otherMatched, opt)),
				}
			}
		}
		var out []*model.SimilarRoute
		for _, c := range j.cands {
			sr := &model.SimilarRoute{
				OtherRouteID:        c.ID,
				SameOnestopID:       true,
				RouteTypeBasicMatch: routeTypeBasicMatch(j.src.RouteType.Int(), c.RouteType.Int()),
			}
			if j.dates != nil {
				a, b := onDays(j.src.ID, j.dates), onDays(c.ID, j.dates)
				pairs := routematch.MatchTrips(a, onDays(c.ID, j.partner), opt)
				otherMatched := len(routematch.MatchTrips(b, onDays(j.src.ID, j.partner), opt))
				rel := model.RouteTripRelationship(routematch.Relate(len(a), len(b), len(pairs), otherMatched, opt))
				sr.Trips, sr.OtherTrips, sr.MatchedTrips, sr.Relationship = ptr(len(a)), ptr(len(b)), ptr(len(pairs)), &rel
				if fc := byFeed[c.FeedOnestopID]; fc != nil {
					feedRel := fc.rel
					sr.FeedTrips, sr.FeedMatchedTrips, sr.FeedRelationship = ptr(fc.trips), ptr(fc.matched), &feedRel
				}
				for _, p := range pairs {
					usedPatterns[p.A.PatternKey] = true
					usedPatterns[p.B.PatternKey] = true
				}
				toAlign = append(toAlign, pending{sr: sr, pairs: pairs})
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
		ret[i] = out
	}

	patternStops, err := f.similarRoutePatternStops(ctx, patternTrips, usedPatterns)
	if err != nil {
		return nil, err
	}
	for _, p := range toAlign {
		p.sr.StopCorrespondences = []*model.RouteStopCorrespondence{}
		for _, corr := range routematch.StopCorrespondences(p.pairs, patternStops, patternStops, opt) {
			p.sr.StopCorrespondences = append(p.sr.StopCorrespondences, &model.RouteStopCorrespondence{
				StopID:       corr.StopID,
				OtherStopID:  corr.OtherStopID,
				MatchedTrips: corr.MatchedTrips,
				Distance:     corr.Distance,
			})
		}
	}
	return ret, nil
}

// The dates to compare a route's candidates on: the filter's service dates as
// given, or its relative date resolved in the route's feed version's timezone
// and the days after it. Neither moves into a fallback week. A feed version
// with no service window resolves a relative date against the clock, as
// Route.patterns does, rather than failing the batch.
func (f *Finder) similarRouteDates(ctx context.Context, src *model.Route, where *model.SimilarRouteFilter) []time.Time {
	if where == nil {
		return nil
	}
	var ret []time.Time
	if len(where.ServiceDates) > 0 {
		for _, d := range where.ServiceDates {
			if d != nil && d.Valid {
				ret = append(ret, model.CivilDate(d.Val))
			}
		}
		return ret
	}
	if where.RelativeDate == nil {
		return nil
	}
	now := model.Now(ctx)
	if fvsw, err := f.FindFeedVersionServiceWindow(ctx, src.FeedVersionID); err == nil && fvsw != nil {
		now = fvsw.NowLocal
	}
	start, err := model.ResolveRelativeDate(*where.RelativeDate, now)
	if err != nil {
		return nil
	}
	days := 1
	if where.Days != nil {
		days = *where.Days
	}
	for i := 0; i < days; i++ {
		ret = append(ret, model.CivilDate(start.AddDate(0, 0, i)))
	}
	return ret
}

// routeTypeBasicMatch reports whether two route types have the same basic type,
// or are both in the rail family: tram or light rail, subway or metro, rail,
// and monorail.
func routeTypeBasicMatch(a, b int) bool {
	ba, bb := tt.BasicRouteType(a), tt.BasicRouteType(b)
	return ba == bb || (railFamily(ba) && railFamily(bb))
}

func railFamily(basic int) bool {
	switch basic {
	case 0, 1, 2, 12:
		return true
	}
	return false
}

// A route's stop pattern: stop_pattern_id is scoped to a feed version, and a
// route belongs to one.
type similarRoutePatternKey struct {
	RouteID       int
	StopPatternID int
}

// The trip a stop pattern's stops are read from: the base trip, holding the
// stop times, of the lowest-numbered trip following the pattern. Its id is
// also the pattern's key in routematch.
type similarRoutePatternTrip struct {
	TripID        int
	FeedVersionID int
}

// A run with its service date, so each comparison can take the dates it asked
// for.
type similarRouteRun struct {
	day   time.Time
	start routematch.TripStart
}

type similarRouteTripDayRow struct {
	TripID      int
	ServiceDate tt.Date
}

type similarRouteTripRow struct {
	TripID         int
	RouteID        int
	StopPatternID  int
	BaseTripID     int
	FeedVersionID  int
	DepartureTime  int
	Lon            float64
	Lat            float64
	AgencyTimezone string
}

type similarRouteFrequencyRow struct {
	TripID      int
	StartTime   int
	EndTime     int
	HeadwaySecs int
}

// similarRouteRuns returns each route's runs on the given days, and the trip
// each stop pattern's stops are read from.
//
// A run starts at its trip's first timed stop. Stop times are read through the
// trip's journey pattern, from its base trip plus its offset, as every other
// stop time query reads them, since an import that deduplicates journey
// patterns stores them only for the base trip. A trip in frequencies.txt runs
// once per headway from start_time up to end_time, at headways of at least
// minExpandedHeadwaySecs, as departure queries expand it. Trips with no timed
// stop, such as flex trips that serve locations or time windows, have no runs.
func (f *Finder) similarRouteRuns(ctx context.Context, routeSet map[int]bool, daySet map[time.Time]bool) (map[int][]similarRouteRun, map[int]similarRoutePatternTrip, error) {
	runs := map[int][]similarRouteRun{}
	patterns := map[int]similarRoutePatternTrip{}
	if len(routeSet) == 0 || len(daySet) == 0 {
		return runs, patterns, nil
	}
	routeIDs := sortedIntKeys(routeSet)
	days := make([]string, 0, len(daySet))
	for d := range daySet {
		days = append(days, d.Format("2006-01-02"))
	}
	sort.Strings(days)

	// The days each trip runs on.
	dq := serviceDatesJoin(
		sq.StatementBuilder.Select("gtfs_trips.id AS trip_id", "service_dates.service_date").
			From("gtfs_trips").
			Where(In("gtfs_trips.route_id", routeIDs)),
		days,
	).OrderBy("gtfs_trips.id", "service_dates.service_date")
	var dayRows []similarRouteTripDayRow
	if err := dbutil.Select(ctx, f.db, dq, &dayRows); err != nil {
		return nil, nil, queryErr(ctx, err)
	}
	tripSet := map[int]bool{}
	for _, r := range dayRows {
		tripSet[r.TripID] = true
	}
	if len(tripSet) == 0 {
		return runs, patterns, nil
	}
	tripIDs := sortedIntKeys(tripSet)

	// Each trip's first timed stop, once, whatever the number of days.
	tq := sq.StatementBuilder.Select(
		"gtfs_trips.id AS trip_id",
		"gtfs_trips.route_id",
		"gtfs_trips.stop_pattern_id",
		"base_trip.id AS base_trip_id",
		"base_trip.feed_version_id",
		"first_st.departure_time + gtfs_trips.journey_pattern_offset AS departure_time",
		"ST_X(first_stop.geometry::geometry) AS lon",
		"ST_Y(first_stop.geometry::geometry) AS lat",
		"gtfs_agencies.agency_timezone",
	).
		From("gtfs_trips").
		Join("gtfs_trips base_trip ON base_trip.trip_id::text = gtfs_trips.journey_pattern_id AND gtfs_trips.feed_version_id = base_trip.feed_version_id").
		JoinClause(`join lateral (
			select st.departure_time, st.stop_id
			from gtfs_stop_times st
			where st.feed_version_id = base_trip.feed_version_id
				and st.trip_id = base_trip.id
				and st.stop_id is not null
				and st.departure_time is not null
			order by st.stop_sequence
			limit 1
		) first_st on true`).
		Join("gtfs_stops first_stop ON first_stop.id = first_st.stop_id").
		Join("gtfs_routes ON gtfs_routes.id = gtfs_trips.route_id").
		Join("gtfs_agencies ON gtfs_agencies.id = gtfs_routes.agency_id").
		Where(In("gtfs_trips.id", tripIDs)).
		OrderBy("gtfs_trips.id")
	var tripRows []similarRouteTripRow
	if err := dbutil.Select(ctx, f.db, tq, &tripRows); err != nil {
		return nil, nil, queryErr(ctx, err)
	}

	// Every frequency row, so a trip whose headways are all under the minimum
	// is still known to be frequency-based, and adds no runs.
	fq := sq.StatementBuilder.Select("trip_id", "start_time", "end_time", "headway_secs").
		From("gtfs_frequencies").
		Where(In("trip_id", tripIDs)).
		OrderBy("trip_id", "start_time")
	var freqRows []similarRouteFrequencyRow
	if err := dbutil.Select(ctx, f.db, fq, &freqRows); err != nil {
		return nil, nil, queryErr(ctx, err)
	}
	freqs := map[int][]similarRouteFrequencyRow{}
	for _, fr := range freqRows {
		freqs[fr.TripID] = append(freqs[fr.TripID], fr)
	}

	locs := map[string]*time.Location{}
	location := func(name string) *time.Location {
		if loc, ok := locs[name]; ok {
			return loc
		}
		loc, err := time.LoadLocation(name)
		if err != nil || name == "" {
			loc = time.UTC
		}
		locs[name] = loc
		return loc
	}
	patternOf := map[similarRoutePatternKey]int{}
	trips := map[int]similarRouteTripRow{}
	for _, t := range tripRows {
		trips[t.TripID] = t
		pk := similarRoutePatternKey{RouteID: t.RouteID, StopPatternID: t.StopPatternID}
		if _, ok := patternOf[pk]; !ok {
			// Rows come in trip id order, so this is the pattern's lowest trip.
			patternOf[pk] = t.BaseTripID
			patterns[t.BaseTripID] = similarRoutePatternTrip{TripID: t.BaseTripID, FeedVersionID: t.FeedVersionID}
		}
	}
	for _, r := range dayRows {
		t, ok := trips[r.TripID]
		if !ok || !r.ServiceDate.Valid {
			continue
		}
		day := model.CivilDate(r.ServiceDate.Val)
		loc := location(t.AgencyTimezone)
		run := routematch.TripStart{
			FirstStop:  tlxy.Point{Lon: t.Lon, Lat: t.Lat},
			PatternKey: patternOf[similarRoutePatternKey{RouteID: t.RouteID, StopPatternID: t.StopPatternID}],
		}
		if fs, ok := freqs[t.TripID]; ok {
			for _, fr := range fs {
				if fr.HeadwaySecs < minExpandedHeadwaySecs {
					continue
				}
				for s := fr.StartTime; s < fr.EndTime; s += fr.HeadwaySecs {
					run.Instant = serviceInstant(day, s, loc)
					runs[t.RouteID] = append(runs[t.RouteID], similarRouteRun{day: day, start: run})
				}
			}
			continue
		}
		run.Instant = serviceInstant(day, t.DepartureTime, loc)
		runs[t.RouteID] = append(runs[t.RouteID], similarRouteRun{day: day, start: run})
	}
	return runs, patterns, nil
}

// serviceInstant is the moment a GTFS time on a service date happens: seconds
// from noon minus 12 hours, local time, which is midnight except on days a
// daylight saving change falls in the night.
func serviceInstant(day time.Time, secs int, loc *time.Location) int64 {
	y, m, d := day.Date()
	return time.Date(y, m, d, 12, 0, 0, 0, loc).Add(-12*time.Hour).Unix() + int64(secs)
}

func sortedIntKeys(set map[int]bool) []int {
	ret := make([]int, 0, len(set))
	for k := range set {
		ret = append(ret, k)
	}
	sort.Ints(ret)
	return ret
}

type similarRoutePatternStopRow struct {
	TripID       int
	StopSequence int
	StopID       int
	Lon          float64
	Lat          float64
}

// similarRoutePatternStops returns the stops, in order, of each pattern in use,
// keyed by the base trip they are read from.
func (f *Finder) similarRoutePatternStops(ctx context.Context, patterns map[int]similarRoutePatternTrip, used map[int]bool) (map[int][]routematch.PatternStop, error) {
	ret := map[int][]routematch.PatternStop{}
	tripSet, fvSet := map[int]bool{}, map[int]bool{}
	for key := range used {
		if p, ok := patterns[key]; ok {
			tripSet[p.TripID] = true
			fvSet[p.FeedVersionID] = true
		}
	}
	if len(tripSet) == 0 {
		return ret, nil
	}
	q := sq.StatementBuilder.Select(
		"st.trip_id",
		"st.stop_sequence",
		"st.stop_id",
		"ST_X(s.geometry::geometry) AS lon",
		"ST_Y(s.geometry::geometry) AS lat",
	).
		From("gtfs_stop_times st").
		Join("gtfs_stops s ON s.id = st.stop_id").
		Where(In("st.feed_version_id", sortedIntKeys(fvSet))).
		Where(In("st.trip_id", sortedIntKeys(tripSet))).
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
