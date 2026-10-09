package gql

import (
	"context"
	"math"
	"sort"

	dataloader "github.com/graph-gophers/dataloader/v7"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/twpayne/go-polyline"
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
				trips = append(trips, tripRun(trip, patternDate(obj)))
			}
		}
		return trips, nil
	}
	// TODO: N+1 query
	trips, err := model.ForContext(ctx).Finder.FindTrips(ctx, resolverCheckLimit(limit), nil, nil, &model.TripFilter{StopPatternID: &obj.StopPatternID, RouteIds: []int{obj.RouteID}})
	return trips, err
}

// patternDate is the date a pattern's trips are counted on, whose runs they are.
func patternDate(obj *model.RouteStopPattern) tt.Date {
	if obj.ServiceDate == nil {
		return tt.Date{}
	}
	return *obj.ServiceDate
}

// Timetable lays the stop times of the trips `count` counted out as grids, a row
// per stop of the pattern and a column per trip, ordered by time at the first
// stop.
func (r *routePatternResolver) Timetable(ctx context.Context, obj *model.RouteStopPattern) (*model.RouteStopPatternTimetable, error) {
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
			cols = append(cols, column{trip: tripRun(trip, patternDate(obj)), sts: sts})
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

	nStops, nTrips := len(header), len(kept)
	ret := &model.RouteStopPatternTimetable{
		StopIds:        make([]int, nStops),
		Trips:          make([]*model.Trip, nTrips),
		DepartureTimes: &model.RouteStopPatternTimetableTimeGrid{Values: newCells[tt.Seconds](nStops, nTrips)},
		ArrivalTimes:   &model.RouteStopPatternTimetableTimeGrid{Values: newCells[tt.Seconds](nStops, nTrips)},
		PickupTypes:    &model.RouteStopPatternTimetableGrid{Values: newCells[tt.Int](nStops, nTrips)},
		DropOffTypes:   &model.RouteStopPatternTimetableGrid{Values: newCells[tt.Int](nStops, nTrips)},
		Timepoints:     &model.RouteStopPatternTimetableGrid{Values: newCells[tt.Int](nStops, nTrips)},
	}
	for i, st := range header {
		ret.StopIds[i] = st.StopID.Int()
	}
	for j, c := range kept {
		ret.Trips[j] = c.trip
		for i, st := range c.sts {
			ret.DepartureTimes.Values[i][j] = st.DepartureTime
			ret.ArrivalTimes.Values[i][j] = st.ArrivalTime
			ret.PickupTypes.Values[i][j] = st.PickupType
			ret.DropOffTypes.Values[i][j] = st.DropOffType
			ret.Timepoints.Values[i][j] = st.Timepoint
		}
	}
	return ret, nil
}

type routePatternGridResolver struct{ *Resolver }

// Delta is the grid as each trip's first non-null value, then each later value
// less it.
func (r *routePatternGridResolver) Delta(ctx context.Context, obj *model.RouteStopPatternTimetableGrid) ([][]*int, error) {
	return gridDeltas(obj.Values), nil
}

// Polyline is each row of Delta as one string, in the polyline algorithm's
// integer encoding: each value less the last one present before it.
func (r *routePatternGridResolver) Polyline(ctx context.Context, obj *model.RouteStopPatternTimetableGrid) ([]string, error) {
	return gridPolylines(gridDeltas(obj.Values)), nil
}

type routePatternTimeGridResolver struct{ *Resolver }

// Delta is the grid as each trip's first non-null time, then each later time
// less it, in seconds.
func (r *routePatternTimeGridResolver) Delta(ctx context.Context, obj *model.RouteStopPatternTimetableTimeGrid) ([][]*int, error) {
	return gridDeltas(obj.Values), nil
}

// Polyline is each row of Delta as one string, encoded as for any other grid.
func (r *routePatternTimeGridResolver) Polyline(ctx context.Context, obj *model.RouteStopPatternTimetableTimeGrid) ([]string, error) {
	return gridPolylines(gridDeltas(obj.Values)), nil
}

// gridCell is a nullable grid value, a tt.Int or tt.Seconds.
type gridCell interface {
	IsValid() bool
	Int() int
}

// gridPolylines encodes each row of a delta grid, '.' for a missing value.
func gridPolylines(rows [][]*int) []string {
	ret := make([]string, len(rows))
	for i, row := range rows {
		var buf []byte
		prev := 0
		for _, v := range row {
			// Encoded characters start at '?', so a '.' cannot be read as one.
			if v == nil {
				buf = append(buf, '.')
				continue
			}
			buf = polyline.EncodeInt(buf, *v-prev)
			prev = *v
		}
		ret[i] = string(buf)
	}
	return ret
}

// gridDeltas keeps each column's first non-null value and makes each later one
// a difference from it.
func gridDeltas[T gridCell](values [][]T) [][]*int {
	ret := make([][]*int, len(values))
	for i, row := range values {
		ret[i] = make([]*int, len(row))
	}
	if len(values) == 0 {
		return ret
	}
	for j := range values[0] {
		var first *int
		for i, row := range values {
			if !row[j].IsValid() {
				continue
			}
			if first == nil {
				first = ptr(row[j].Int())
				ret[i][j] = first
				continue
			}
			ret[i][j] = ptr(row[j].Int() - *first)
		}
	}
	return ret
}

// newCells is a grid of zero values, rows by cols.
func newCells[T any](rows, cols int) [][]T {
	ret := make([][]T, rows)
	for i := range ret {
		ret[i] = make([]T, cols)
	}
	return ret
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
