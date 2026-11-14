package dbfinder

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/interline-io/transitland-lib/server/model"
)

// StopLocator finds the stops that a Finder's stop buffers read. NewFinder
// makes a Finder its own StopLocator; another one lets the Finder buffer stops
// that its tables don't hold.
//
// A census search still checks each located stop's feed version in the
// database, and leaves out a stop whose feed version isn't imported there or
// isn't visible to the caller.
type StopLocator interface {
	StopsByIDs(ctx context.Context, ids []int) ([]*model.Stop, []error)
	StopsByRouteIDs(ctx context.Context, limit *int, where *model.StopFilter, keys []int) ([][]*model.Stop, error)
	StopsByAgencyIDs(ctx context.Context, limit *int, after *model.Cursor, where *model.AgencyStopFilter, keys []int) ([][]*model.Stop, error)
}

// locatedStops are the stops a StopLocator found.
type locatedStops struct {
	stops []*model.Stop
}

// ToSql selects the located stops with the id, feed_version_id and geometry
// columns of gtfs_stops, leaving out stops without a location.
func (s *locatedStops) ToSql() (string, []any, error) {
	type row struct {
		ID            int     `json:"id"`
		FeedVersionID int     `json:"feed_version_id"`
		Lon           float64 `json:"lon"`
		Lat           float64 `json:"lat"`
	}
	rows := []row{}
	for _, stop := range s.stops {
		if stop == nil || !stop.Geometry.Valid {
			continue
		}
		rows = append(rows, row{ID: stop.ID, FeedVersionID: stop.FeedVersionID, Lon: stop.Geometry.X(), Lat: stop.Geometry.Y()})
	}
	// The rows reach the query as JSON objects in one parameter, which pairs
	// each value with its column.
	data, err := json.Marshal(rows)
	if err != nil {
		return "", nil, fmt.Errorf("encoding located stops: %w", err)
	}
	return "SELECT s.id, s.feed_version_id, ST_SetSRID(ST_MakePoint(s.lon, s.lat), 4326)::geography AS geometry FROM json_to_recordset(?::json) AS s(id bigint, feed_version_id bigint, lon float8, lat float8)", []any{string(data)}, nil
}

// locateStops finds the stops with these ids through the stop locator.
func (f *Finder) locateStops(ctx context.Context, ids []int) (*locatedStops, error) {
	// A stop named twice is buffered once.
	stops, errs := f.StopLocator.StopsByIDs(ctx, slices.Compact(slices.Sorted(slices.Values(ids))))
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return &locatedStops{stops: stops}, nil
}

// locateEntityStops finds a route's or agency's stops through the stop locator.
func (f *Finder) locateEntityStops(ctx context.Context, entityType string, entityId int) (*locatedStops, error) {
	var groups [][]*model.Stop
	var err error
	switch entityType {
	case "route":
		groups, err = f.StopLocator.StopsByRouteIDs(ctx, nil, nil, []int{entityId})
	case "agency":
		groups, err = f.StopLocator.StopsByAgencyIDs(ctx, nil, nil, nil, []int{entityId})
	default:
		return nil, fmt.Errorf("no stops for entity type %q", entityType)
	}
	if err != nil {
		return nil, err
	}
	if len(groups) != 1 {
		return nil, fmt.Errorf("stop locator returned %d groups of stops for one %s", len(groups), entityType)
	}
	return &locatedStops{stops: groups[0]}, nil
}
