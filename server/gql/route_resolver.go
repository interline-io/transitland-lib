package gql

import (
	"context"
	"math"
	"sort"

	dataloader "github.com/graph-gophers/dataloader/v7"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tt"
)

// ROUTE

type routeResolver struct{ *Resolver }

func (r *routeResolver) Cursor(ctx context.Context, obj *model.Route) (*model.Cursor, error) {
	c := model.NewCursor(obj.FeedVersionID, obj.ID)
	return &c, nil
}

// RouteTypeBasic returns the basic GTFS route type this route's raw one is a kind
// of, so a caller grouping by mode does not have to carry the hierarchy itself.
func (r *routeResolver) RouteTypeBasic(ctx context.Context, obj *model.Route) (int, error) {
	return tt.BasicRouteType(obj.RouteType.Int()), nil
}

func (r *routeResolver) Geometry(ctx context.Context, obj *model.Route) (*tt.Geometry, error) {
	if obj.Geometry.Valid {
		return &obj.Geometry, nil
	}
	// Defer geometry loading
	geoms, err := LoaderFor(ctx).RouteGeometriesByRouteIDs.Load(ctx, routeGeometryLoaderParam{RouteID: obj.ID})()
	if err != nil {
		return nil, err
	}
	if len(geoms) > 0 {
		return geoms[0].CombinedGeometry, nil
	}
	return nil, nil
}

func (r *routeResolver) Geometries(ctx context.Context, obj *model.Route, limit *int) ([]*model.RouteGeometry, error) {
	return LoaderFor(ctx).RouteGeometriesByRouteIDs.Load(ctx, routeGeometryLoaderParam{RouteID: obj.ID, Limit: resolverCheckLimit(limit)})()
}

func (r *routeResolver) Trips(ctx context.Context, obj *model.Route, limit *int, where *model.TripFilter) ([]*model.Trip, error) {
	return LoaderFor(ctx).TripsByRouteIDs.Load(ctx, tripLoaderParam{RouteID: obj.ID, FeedVersionID: obj.FeedVersionID, Limit: resolverCheckLimitMax(limit, RESOLVER_TRIP_MAXLIMIT), Where: where})()
}

func (r *routeResolver) Agency(ctx context.Context, obj *model.Route) (*model.Agency, error) {
	return LoaderFor(ctx).AgenciesByIDs.Load(ctx, obj.AgencyID.Int())()
}

func (r *routeResolver) FeedVersion(ctx context.Context, obj *model.Route) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

func (r *routeResolver) Stops(ctx context.Context, obj *model.Route, limit *int, where *model.StopFilter) ([]*model.Stop, error) {
	return LoaderFor(ctx).StopsByRouteIDs.Load(ctx, stopLoaderParam{RouteID: obj.ID, Limit: resolverCheckLimit(limit), Where: where})()
}

func (r *routeResolver) RouteStops(ctx context.Context, obj *model.Route, limit *int) ([]*model.RouteStop, error) {
	return LoaderFor(ctx).RouteStopsByRouteIDs.Load(ctx, routeStopLoaderParam{RouteID: obj.ID, Limit: resolverCheckLimit(limit)})()
}

func (r *routeResolver) Headways(ctx context.Context, obj *model.Route, limit *int) ([]*model.RouteHeadway, error) {
	return LoaderFor(ctx).RouteHeadwaysByRouteIDs.Load(ctx, routeHeadwayLoaderParam{RouteID: obj.ID, Limit: resolverCheckLimit(limit)})()
}

func (r *routeResolver) RouteStopBuffer(ctx context.Context, obj *model.Route, radius *float64) (*model.RouteStopBuffer, error) {
	// TODO: remove n+1 (which is tricky, what if multiple radius specified in different parts of query)
	ents, err := model.ForContext(ctx).Finder.RouteStopBuffer(ctx, nil, radius, obj.ID)
	if err != nil {
		return nil, err
	}
	if len(ents) > 0 {
		return ents[0], nil
	}
	return nil, nil
}

func (r *routeResolver) Alerts(ctx context.Context, obj *model.Route, active *bool, limit *int, includeModes *bool, includeTrips *bool) ([]*model.Alert, error) {
	return model.ForContext(ctx).RTFinder.FindAlertsForRoute(ctx, obj, resolverCheckLimit(limit), active, includeModes != nil && *includeModes, includeTrips != nil && *includeTrips), nil
}

func (r *routeResolver) Patterns(ctx context.Context, obj *model.Route, where *model.RouteStopPatternFilter) ([]*model.RouteStopPattern, error) {
	return LoaderFor(ctx).RouteStopPatternsByRouteIDs.Load(ctx, routeStopPatternLoaderParam{
		FeedVersionID: obj.FeedVersionID,
		RouteID:       obj.ID,
		Where:         where,
	})()
}

func (r *routeResolver) RouteAttribute(ctx context.Context, obj *model.Route) (*model.RouteAttribute, error) {
	return LoaderFor(ctx).RouteAttributesByRouteIDs.Load(ctx, obj.ID)()
}

func (r *routeResolver) SegmentPatterns(ctx context.Context, obj *model.Route, limit *int, where *model.SegmentPatternFilter) ([]*model.SegmentPattern, error) {
	return LoaderFor(ctx).SegmentPatternsByRouteIDs.Load(ctx, segmentPatternLoaderParam{RouteID: obj.ID, Where: where, Limit: resolverCheckLimitMax(limit, RESOLVER_SEGMENT_MAXLIMIT)})()
}

func (r *routeResolver) Segments(ctx context.Context, obj *model.Route, limit *int, where *model.SegmentFilter) ([]*model.Segment, error) {
	return LoaderFor(ctx).SegmentsByRouteIDs.Load(ctx, segmentLoaderParam{RouteID: obj.ID, Where: where, Limit: resolverCheckLimitMax(limit, RESOLVER_SEGMENT_MAXLIMIT)})()
}

// ROUTE HEADWAYS

type routeHeadwayResolver struct{ *Resolver }

func (r *routeHeadwayResolver) Stop(ctx context.Context, obj *model.RouteHeadway) (*model.Stop, error) {
	return LoaderFor(ctx).StopsByIDs.Load(ctx, obj.SelectedStopID)()
}

func (r *routeHeadwayResolver) Departures(ctx context.Context, obj *model.RouteHeadway) ([]*tt.Seconds, error) {
	var ret []*tt.Seconds
	for _, v := range obj.DepartureInts.Val {
		w := tt.NewSeconds(int(v))
		ret = append(ret, &w)
	}
	return ret, nil
}

// ROUTE STOP

type routeStopResolver struct{ *Resolver }

func (r *routeStopResolver) Route(ctx context.Context, obj *model.RouteStop) (*model.Route, error) {
	return LoaderFor(ctx).RoutesByIDs.Load(ctx, obj.RouteID)()
}

func (r *routeStopResolver) Stop(ctx context.Context, obj *model.RouteStop) (*model.Stop, error) {
	return LoaderFor(ctx).StopsByIDs.Load(ctx, obj.StopID)()
}

func (r *routeStopResolver) Agency(ctx context.Context, obj *model.RouteStop) (*model.Agency, error) {
	return LoaderFor(ctx).AgenciesByIDs.Load(ctx, obj.AgencyID)()
}

// ROUTE PATTERN

type routePatternResolver struct{ *Resolver }

// The trip the pattern was derived from, already known from the aggregate, so this
// is a batched lookup by id rather than a search for a trip matching the pattern.
func (r *routePatternResolver) RepresentativeTrip(ctx context.Context, obj *model.RouteStopPattern) (*model.Trip, error) {
	if obj.RepresentativeTripID == 0 {
		return nil, nil
	}
	return LoaderFor(ctx).TripsByIDs.Load(ctx, obj.RepresentativeTripID)()
}

func (r *routePatternResolver) Trips(ctx context.Context, obj *model.RouteStopPattern, limit *int) ([]*model.Trip, error) {
	// On a date, the trips counted rather than every trip of the pattern.
	if len(obj.TripIDs.Val) > 0 {
		ids := obj.TripIDs.Val
		if n := *resolverCheckLimit(limit); n < len(ids) {
			ids = ids[:n]
		}
		var loads []dataloader.Thunk[*model.Trip]
		for _, id := range ids {
			loads = append(loads, LoaderFor(ctx).TripsByIDs.Load(ctx, int(id)))
		}
		var trips []*model.Trip
		for _, load := range loads {
			trip, err := load()
			if err != nil {
				return nil, err
			}
			if trip != nil {
				trips = append(trips, trip)
			}
		}
		return trips, nil
	}
	// TODO: N+1 query
	trips, err := model.ForContext(ctx).Finder.FindTrips(ctx, resolverCheckLimit(limit), nil, nil, &model.TripFilter{StopPatternID: &obj.StopPatternID, RouteIds: []int{obj.RouteID}})
	return trips, err
}

// Departures lays the stop times of the trips `count` counted out as grids, a row
// per stop of the pattern and a column per trip, ordered by time at the first
// stop.
func (r *routePatternResolver) Departures(ctx context.Context, obj *model.RouteStopPattern) (*model.RouteStopPatternDepartures, error) {
	if len(obj.TripIDs.Val) == 0 {
		return nil, nil
	}

	type column struct {
		trip *model.Trip
		sts  []*model.StopTime
	}
	// Every load is asked for before any is read, so the loaders batch them.
	var tripLoads []dataloader.Thunk[*model.Trip]
	var stopTimeLoads []dataloader.Thunk[[]*model.StopTime]
	for _, id := range obj.TripIDs.Val {
		tripLoads = append(tripLoads, LoaderFor(ctx).TripsByIDs.Load(ctx, int(id)))
		stopTimeLoads = append(stopTimeLoads, LoaderFor(ctx).StopTimesByTripIDs.Load(ctx, tripStopTimeLoaderParam{
			FeedVersionID: obj.FeedVersionID,
			TripID:        int(id),
			Limit:         ptr(RESOLVER_PATTERN_MAXLIMIT),
		}))
	}
	// Flex trips run to locations or within time windows rather than at set
	// times, so they have no cells. A pattern can mix them with fixed trips.
	var cols []column
	for i := range tripLoads {
		trip, err := tripLoads[i]()
		if err != nil {
			return nil, err
		}
		sts, err := stopTimeLoads[i]()
		if err != nil {
			return nil, err
		}
		if trip != nil && len(sts) > 0 && !isFlex(sts) {
			cols = append(cols, column{trip: trip, sts: sts})
		}
	}
	if len(cols) == 0 {
		return nil, nil
	}

	// The pattern's stops are those of the trip storing the most, the lowest ID on
	// a tie. Trips share a pattern by the stops they were imported with, and the
	// import can drop a trip's invalid rows while keeping the trip.
	header := cols[0].sts
	for _, c := range cols[1:] {
		if len(c.sts) > len(header) {
			header = c.sts
		}
	}
	var kept []column
	for _, c := range cols {
		if sameStops(c.sts, header) {
			kept = append(kept, c)
		}
	}
	// A trip with no time at the first stop sorts last rather than at midnight.
	firstTime := func(c column) int {
		st := c.sts[0]
		if st.DepartureTime.Valid {
			return st.DepartureTime.Int()
		}
		if st.ArrivalTime.Valid {
			return st.ArrivalTime.Int()
		}
		return math.MaxInt
	}
	sort.Slice(kept, func(a, b int) bool {
		ta, tb := firstTime(kept[a]), firstTime(kept[b])
		if ta != tb {
			return ta < tb
		}
		return kept[a].trip.ID < kept[b].trip.ID
	})

	ret := &model.RouteStopPatternDepartures{
		StopIds:        make([]int, len(header)),
		Trips:          make([]*model.Trip, len(kept)),
		DepartureTimes: make([][]tt.Seconds, len(header)),
		ArrivalTimes:   make([][]tt.Seconds, len(header)),
		PickupTypes:    make([][]tt.Int, len(header)),
		DropOffTypes:   make([][]tt.Int, len(header)),
		Timepoints:     make([][]tt.Int, len(header)),
	}
	for i, st := range header {
		ret.StopIds[i] = st.StopID.Int()
		ret.DepartureTimes[i] = make([]tt.Seconds, len(kept))
		ret.ArrivalTimes[i] = make([]tt.Seconds, len(kept))
		ret.PickupTypes[i] = make([]tt.Int, len(kept))
		ret.DropOffTypes[i] = make([]tt.Int, len(kept))
		ret.Timepoints[i] = make([]tt.Int, len(kept))
	}
	for j, c := range kept {
		ret.Trips[j] = c.trip
		for i, st := range c.sts {
			ret.DepartureTimes[i][j] = st.DepartureTime
			ret.ArrivalTimes[i][j] = st.ArrivalTime
			ret.PickupTypes[i][j] = st.PickupType
			ret.DropOffTypes[i][j] = st.DropOffType
			ret.Timepoints[i][j] = st.Timepoint
		}
	}
	return ret, nil
}

// isFlex reports whether stop times run to locations or within time windows.
func isFlex(sts []*model.StopTime) bool {
	v := make([]gtfs.StopTime, len(sts))
	for i, st := range sts {
		v[i] = st.StopTime
	}
	return gtfs.CheckFlexStopTimes(v).IsFlexTrip()
}

// sameStops reports whether two runs of stop times call at the same stops in
// the same order.
func sameStops(a, b []*model.StopTime) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].StopID.Val != b[i].StopID.Val {
			return false
		}
	}
	return true
}

// func (r *routePatternResolver) Stops(ctx context.Context, obj *model.RouteStopPattern) ([]*model.Stop, error) {
// 	return nil, nil
// }
