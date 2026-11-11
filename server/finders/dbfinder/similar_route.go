package dbfinder

import (
	"cmp"
	"context"
	"maps"
	"slices"
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
func (f *Finder) SimilarRoutesByRouteIDs(ctx context.Context, limit *int, query model.SimilarRouteQuery, keys []int) ([][]*model.SimilarRoute, error) {
	ret := make([][]*model.SimilarRoute, len(keys))
	for i := range ret {
		ret[i] = []*model.SimilarRoute{}
	}
	jobs, involved, allDays, err := f.similarRouteJobs(ctx, query.Where, keys)
	if err != nil {
		return nil, err
	}
	runs, patternTrips, err := f.similarRouteRuns(ctx, involved, allDays)
	if err != nil {
		return nil, err
	}

	// Stop correspondences wait until every pairing is known, so only patterns
	// that matched load their stops.
	opt := routematch.DefaultOptions()
	var toAlign []similarRouteCompared
	usedPatterns := map[int]bool{}
	for i, j := range jobs {
		if j == nil {
			continue
		}
		rows := compareSimilarRoutes(j, runs, opt)
		if n := finderCheckLimit(limit); uint64(len(rows)) > n {
			rows = rows[:n]
		}
		for _, row := range rows {
			ret[i] = append(ret[i], row.sr)
			if query.StopCorrespondences && j.dates != nil {
				toAlign = append(toAlign, row)
				for _, p := range row.pairs {
					usedPatterns[p.A.PatternKey] = true
					usedPatterns[p.B.PatternKey] = true
				}
			}
		}
	}
	if len(toAlign) == 0 {
		return ret, nil
	}
	patternStops, err := f.similarRoutePatternStops(ctx, patternTrips, usedPatterns)
	if err != nil {
		return nil, err
	}
	for _, row := range toAlign {
		row.sr.StopCorrespondences = []*model.RouteStopCorrespondence{}
		for _, corr := range routematch.StopCorrespondences(row.pairs, patternStops, opt) {
			row.sr.StopCorrespondences = append(row.sr.StopCorrespondences, &model.RouteStopCorrespondence{
				StopID:       corr.StopID,
				OtherStopID:  corr.OtherStopID,
				MatchedTrips: corr.MatchedTrips,
				Distance:     corr.Distance,
			})
		}
	}
	return ret, nil
}

// A route with its candidates and the dates to compare them on. Each route's
// trips are compared over its own dates; a candidate's runs on the days either
// side count as partners for runs past midnight, but not toward its trips.
// Without dates, the candidates are listed uncompared.
type similarRouteJob struct {
	src     *model.Route
	cands   []*model.Route
	dates   map[time.Time]bool
	partner map[time.Time]bool
}

// similarRouteJobs pairs each route with its candidates, index for index with
// keys (nil for a route with none), and returns the routes and days whose runs
// the comparisons need.
func (f *Finder) similarRouteJobs(ctx context.Context, where *model.SimilarRouteFilter, keys []int) ([]*similarRouteJob, map[int]bool, map[time.Time]bool, error) {
	jobs := make([]*similarRouteJob, len(keys))
	involved := map[int]bool{}
	allDays := map[time.Time]bool{}
	sources, errs := f.RoutesByIDs(ctx, keys)
	for _, err := range errs {
		if err != nil {
			return nil, nil, nil, err
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
		return jobs, involved, allDays, nil
	}
	// Active routes only: the candidates are what global queries return now.
	cands, err := f.FindRoutes(ctx, nil, nil, nil, &model.RouteFilter{OnestopIds: osids})
	if err != nil {
		return nil, nil, nil, err
	}
	candsByOsid := map[string][]*model.Route{}
	for _, c := range cands {
		if c.OnestopID != nil {
			candsByOsid[*c.OnestopID] = append(candsByOsid[*c.OnestopID], c)
		}
	}
	otherFeedsOnly := where == nil || where.OtherFeedsOnly == nil || *where.OtherFeedsOnly
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
		j := &similarRouteJob{src: src, cands: cs}
		jobs[i] = j
		// A relative date resolves in the route's feed version's timezone. A
		// feed version with no service window resolves it against the clock, as
		// Route.patterns does, rather than failing the batch.
		now := model.Now(ctx)
		if where != nil && where.RelativeDate != nil && len(where.ServiceDates) == 0 {
			if fvsw, err := f.FindFeedVersionServiceWindow(ctx, src.FeedVersionID); err == nil && fvsw != nil {
				now = fvsw.NowLocal
			}
		}
		dates := model.SimilarRouteDates(where, now)
		if len(dates) == 0 {
			continue
		}
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
	return jobs, involved, allDays, nil
}

// A candidate as compared, with the matched trip pairs its stop
// correspondences come from.
type similarRouteCompared struct {
	sr    *model.SimilarRoute
	pairs []routematch.Pair
}

// compareSimilarRoutes compares a job's route with each candidate, and with all
// of each candidate feed's rows together, since a feed can split one route into
// rows that no single one covers. Most matched trips first, then by id, so a
// limit keeps the likeliest copies.
func compareSimilarRoutes(j *similarRouteJob, runs map[int][]similarRouteRun, opt routematch.Options) []similarRouteCompared {
	onDays := func(routeID int, days map[time.Time]bool) []routematch.TripStart {
		var out []routematch.TripStart
		for _, r := range runs[routeID] {
			if days[r.day] {
				out = append(out, r.start)
			}
		}
		return out
	}
	var a, aPartner []routematch.TripStart
	if j.dates != nil {
		a, aPartner = onDays(j.src.ID, j.dates), onDays(j.src.ID, j.partner)
	}
	type feedRows struct {
		b, bPartner []routematch.TripStart
		rows        []routematch.Comparison
	}
	byFeed := map[string]*feedRows{}
	out := make([]similarRouteCompared, 0, len(j.cands))
	for _, c := range j.cands {
		sr := &model.SimilarRoute{
			OtherRouteID:        c.ID,
			SameOnestopID:       true,
			RouteTypeBasicMatch: tt.BasicRouteTypesMatch(j.src.RouteType.Int(), c.RouteType.Int()),
		}
		var res routematch.Comparison
		if j.dates != nil {
			b, bPartner := onDays(c.ID, j.dates), onDays(c.ID, j.partner)
			res = routematch.Compare(a, aPartner, b, bPartner, opt)
			sr.Trips, sr.OtherTrips, sr.MatchedTrips, sr.Relationship = ptr(res.Trips), ptr(res.OtherTrips), ptr(len(res.Pairs)), relationshipOf(res)
			fr := byFeed[c.FeedOnestopID]
			if fr == nil {
				fr = &feedRows{}
				byFeed[c.FeedOnestopID] = fr
			}
			fr.b, fr.bPartner, fr.rows = append(fr.b, b...), append(fr.bPartner, bPartner...), append(fr.rows, res)
		}
		out = append(out, similarRouteCompared{sr: sr, pairs: res.Pairs})
	}
	if j.dates != nil {
		// A feed with one row compares as that row did.
		feedResult := map[string]routematch.Comparison{}
		for feed, fr := range byFeed {
			if len(fr.rows) == 1 {
				feedResult[feed] = fr.rows[0]
			} else {
				feedResult[feed] = routematch.Compare(a, aPartner, fr.b, fr.bPartner, opt)
			}
		}
		for i, c := range j.cands {
			fres := feedResult[c.FeedOnestopID]
			out[i].sr.FeedTrips, out[i].sr.FeedMatchedTrips, out[i].sr.FeedRelationship = ptr(fres.OtherTrips), ptr(len(fres.Pairs)), relationshipOf(fres)
		}
	}
	matched := func(sr *model.SimilarRoute) int {
		if sr.MatchedTrips == nil {
			return -1
		}
		return *sr.MatchedTrips
	}
	slices.SortStableFunc(out, func(x, y similarRouteCompared) int {
		return cmp.Or(cmp.Compare(matched(y.sr), matched(x.sr)), cmp.Compare(x.sr.OtherRouteID, y.sr.OtherRouteID))
	})
	return out
}

func relationshipOf(c routematch.Comparison) *model.RouteTripRelationship {
	rel := model.RouteTripRelationship(c.Relationship)
	return &rel
}

// A route's stop pattern: stop_pattern_id is scoped to a feed version, and a
// route belongs to one.
type similarRoutePatternKey struct {
	RouteID       int
	StopPatternID int
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
// each stop pattern's stops are read from, with its feed version: the base
// trip, holding the stop times, of the lowest-numbered trip following the
// pattern. That trip's id is also the pattern's key in routematch.
//
// A run starts at its trip's first timed stop. Stop times are read through the
// trip's journey pattern, from its base trip plus its offset, as every other
// stop time query reads them, since an import that deduplicates journey
// patterns stores them only for the base trip. A trip in frequencies.txt runs
// once per headway from start_time up to end_time, at headways of at least
// minExpandedHeadwaySecs, as departure queries expand it. Trips with no timed
// stop, such as flex trips that serve locations or time windows, have no runs.
func (f *Finder) similarRouteRuns(ctx context.Context, routeSet map[int]bool, daySet map[time.Time]bool) (map[int][]similarRouteRun, map[int]int, error) {
	runs := map[int][]similarRouteRun{}
	patterns := map[int]int{}
	if len(routeSet) == 0 || len(daySet) == 0 {
		return runs, patterns, nil
	}
	routeIDs := slices.Sorted(maps.Keys(routeSet))
	days := make([]string, 0, len(daySet))
	for d := range daySet {
		days = append(days, d.Format("2006-01-02"))
	}
	slices.Sort(days)

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
	tripIDs := slices.Sorted(maps.Keys(tripSet))

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

	patternOf := map[similarRoutePatternKey]int{}
	trips := map[int]similarRouteTripRow{}
	locs := map[int]*time.Location{}
	for _, t := range tripRows {
		trips[t.TripID] = t
		loc, err := f.fvslCache.loadLocation(t.AgencyTimezone)
		if err != nil {
			loc = time.UTC
		}
		locs[t.TripID] = loc
		pk := similarRoutePatternKey{RouteID: t.RouteID, StopPatternID: t.StopPatternID}
		if _, ok := patternOf[pk]; !ok {
			// Rows come in trip id order, so this is the pattern's lowest trip.
			patternOf[pk] = t.BaseTripID
			patterns[t.BaseTripID] = t.FeedVersionID
		}
	}
	for _, r := range dayRows {
		t, ok := trips[r.TripID]
		if !ok || !r.ServiceDate.Valid {
			continue
		}
		// GTFS times count from noon less twelve hours on the service date, in
		// the agency's timezone.
		day := model.CivilDate(r.ServiceDate.Val)
		dayStart := tt.ServiceDayStart(r.ServiceDate, locs[t.TripID]).Unix()
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
					run.Instant = dayStart + int64(s)
					runs[t.RouteID] = append(runs[t.RouteID], similarRouteRun{day: day, start: run})
				}
			}
			continue
		}
		run.Instant = dayStart + int64(t.DepartureTime)
		runs[t.RouteID] = append(runs[t.RouteID], similarRouteRun{day: day, start: run})
	}
	return runs, patterns, nil
}

type similarRoutePatternStopRow struct {
	TripID       int
	StopSequence int
	StopID       int
	Lon          float64
	Lat          float64
}

// similarRoutePatternStops returns the stops, in order, of each pattern in use,
// keyed by the base trip they are read from. patterns gives each base trip's
// feed version.
func (f *Finder) similarRoutePatternStops(ctx context.Context, patterns map[int]int, used map[int]bool) (map[int][]routematch.PatternStop, error) {
	ret := map[int][]routematch.PatternStop{}
	tripSet, fvSet := map[int]bool{}, map[int]bool{}
	for tripID := range used {
		if fvid, ok := patterns[tripID]; ok {
			tripSet[tripID] = true
			fvSet[fvid] = true
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
		Where(In("st.feed_version_id", slices.Sorted(maps.Keys(fvSet)))).
		Where(In("st.trip_id", slices.Sorted(maps.Keys(tripSet)))).
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
