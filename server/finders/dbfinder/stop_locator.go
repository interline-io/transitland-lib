package dbfinder

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/interline-io/transitland-lib/server/model"
	sq "github.com/irees/squirrel"
)

// StopLocator finds the stops that stop buffers read: the stops a census stop
// buffer names, the stops of the routes and agencies whose census geographies
// are requested, and a route's stops for its stop buffer.
//
// A Finder reads these stops from gtfs_stops and tl_route_stops unless it has a
// StopLocator, which lets a Finder serve stops that those tables don't hold.
type StopLocator interface {
	FindStops(ctx context.Context, limit *int, after *model.Cursor, ids []int, where *model.StopFilter) ([]*model.Stop, error)
	StopsByRouteIDs(ctx context.Context, limit *int, where *model.StopFilter, keys []int) ([][]*model.Stop, error)
	StopsByAgencyIDs(ctx context.Context, limit *int, after *model.Cursor, where *model.AgencyStopFilter, keys []int) ([][]*model.Stop, error)
}

// SetStopLocator makes the finder read the stops that stop buffers name from
// locator.
func (f *Finder) SetStopLocator(locator StopLocator) {
	f.stopLocator = locator
}

// locatedStops are the stops a StopLocator found, as the JSON rows that
// selectStops reads.
type locatedStops struct {
	rows []byte
}

// locatedStop is a located stop's columns of gtfs_stops, with its location as
// coordinates.
type locatedStop struct {
	ID            int     `json:"id"`
	FeedVersionID int     `json:"feed_version_id"`
	Lon           float64 `json:"lon"`
	Lat           float64 `json:"lat"`
}

// newLocatedStops encodes the stops that have a location, since a stop without
// one has no point to buffer.
func newLocatedStops(stops []*model.Stop) (*locatedStops, error) {
	rows := []locatedStop{}
	for _, stop := range stops {
		row := locatedStop{ID: stop.ID, FeedVersionID: stop.FeedVersionID}
		switch {
		case stop.Geometry.Valid:
			row.Lon, row.Lat = stop.Geometry.X(), stop.Geometry.Y()
		case stop.StopLon.Valid && stop.StopLat.Valid:
			row.Lon, row.Lat = stop.StopLon.Val, stop.StopLat.Val
		default:
			continue
		}
		rows = append(rows, row)
	}
	data, err := json.Marshal(rows)
	if err != nil {
		return nil, fmt.Errorf("encoding located stops: %w", err)
	}
	return &locatedStops{rows: data}, nil
}

// selectStops selects the stops with the id, feed_version_id and geometry
// columns of gtfs_stops.
func (s *locatedStops) selectStops() sq.Sqlizer {
	return sq.Expr(
		"SELECT s.id, s.feed_version_id, ST_SetSRID(ST_MakePoint(s.lon, s.lat), 4326)::geography AS geometry FROM json_to_recordset(?::json) AS s(id bigint, feed_version_id bigint, lon float8, lat float8)",
		string(s.rows),
	)
}

// locateStops finds the stops with these ids through the stop locator, or
// returns nil when the finder has none and reads gtfs_stops.
func (f *Finder) locateStops(ctx context.Context, ids []int) (*locatedStops, error) {
	if f.stopLocator == nil {
		return nil, nil
	}
	stops, err := f.stopLocator.FindStops(ctx, nil, nil, ids, nil)
	if err != nil {
		return nil, err
	}
	return newLocatedStops(stops)
}

// locateStopBuffer finds the stops a census location filter's stop buffer names
// through the stop locator, or returns nil when the finder reads gtfs_stops.
func (f *Finder) locateStopBuffer(ctx context.Context, loc *model.CensusDatasetGeographyLocationFilter) (*locatedStops, error) {
	if loc == nil || loc.StopBuffer == nil || len(loc.StopBuffer.StopIds) == 0 {
		return nil, nil
	}
	return f.locateStops(ctx, loc.StopBuffer.StopIds)
}

// locateEntityStops finds each route's or agency's stops through the stop
// locator, as the stop ids and located stops of each.
func (f *Finder) locateEntityStops(ctx context.Context, entityType string, entityIds []int) ([][]int, []*locatedStops, error) {
	var groups [][]*model.Stop
	var err error
	switch entityType {
	case "route":
		groups, err = f.stopLocator.StopsByRouteIDs(ctx, nil, nil, entityIds)
	case "agency":
		groups, err = f.stopLocator.StopsByAgencyIDs(ctx, nil, nil, nil, entityIds)
	default:
		return nil, nil, fmt.Errorf("no stops for entity type %q", entityType)
	}
	if err != nil {
		return nil, nil, err
	}
	stopIds := make([][]int, len(entityIds))
	located := make([]*locatedStops, len(entityIds))
	for i := range entityIds {
		var stops []*model.Stop
		if i < len(groups) {
			stops = groups[i]
		}
		for _, stop := range stops {
			stopIds[i] = append(stopIds[i], stop.ID)
		}
		if located[i], err = newLocatedStops(stops); err != nil {
			return nil, nil, err
		}
	}
	return stopIds, located, nil
}
