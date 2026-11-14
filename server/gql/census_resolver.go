package gql

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/interline-io/transitland-lib/server/model"
)

////////////////////////// CENSUS RESOLVERS

type censusDatasetResolver struct{ *Resolver }

func (r *censusDatasetResolver) Geographies(ctx context.Context, obj *model.CensusDataset, limit *int, where *model.CensusDatasetGeographyFilter) ([]*model.CensusGeography, error) {
	return LoaderFor(ctx).CensusGeographiesByDatasetIDs.Load(ctx, censusDatasetGeographyLoaderParam{DatasetID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_CENSUS_MAXLIMIT), Where: where, Select: censusGeographySelect(ctx)})()
}

func (r *censusDatasetResolver) Sources(ctx context.Context, obj *model.CensusDataset, limit *int, where *model.CensusSourceFilter) ([]*model.CensusSource, error) {
	return LoaderFor(ctx).CensusSourcesByDatasetIDs.Load(ctx, censusSourceLoaderParam{DatasetID: obj.ID, Limit: resolverCheckLimit(limit), Where: where})()
}

func (r *censusDatasetResolver) Tables(ctx context.Context, obj *model.CensusDataset, limit *int, where *model.CensusTableFilter) ([]*model.CensusTable, error) {
	return LoaderFor(ctx).CensusTablesByDatasetIDs.Load(ctx, censusTableLoaderParam{DatasetID: obj.ID, Limit: resolverCheckLimit(limit), Where: where})()
}

func (r *censusDatasetResolver) Layers(ctx context.Context, obj *model.CensusDataset) (ret []*model.CensusLayer, err error) {
	return LoaderFor(ctx).CensusDatasetLayersByDatasetIDs.Load(ctx, obj.ID)()
}

func (r *censusDatasetResolver) ValuesRelay(ctx context.Context, obj *model.CensusDataset, first *int, after *string, where *model.CensusDatasetValueFilter) (*model.CensusValueConnection, error) {
	cfg := model.ForContext(ctx)

	// Decode cursor if provided
	var cursor model.CensusCursor
	var err error
	if after != nil && *after != "" {
		cursor, err = model.DecodeCensusCursor(*after)
		if err != nil {
			return nil, err
		}
	}

	// Fetch one extra to determine if there's a next page
	limit := *resolverCheckLimit(first)
	fetchLimit := limit + 1

	values, err := cfg.Finder.FindCensusValuesByDatasetID(ctx, &fetchLimit, cursor, obj.ID, where)
	if err != nil {
		return nil, err
	}

	// Determine if there are more results
	hasNextPage := len(values) > limit
	if hasNextPage {
		values = values[:limit]
	}

	// Build edges
	edges := make([]*model.CensusValueEdge, len(values))
	for i, value := range values {
		valueCursor := model.NewCensusCursor(value.Geoid, value.TableID)
		edges[i] = &model.CensusValueEdge{
			Node:   value,
			Cursor: valueCursor.Encode(),
		}
	}

	// Build page info
	pageInfo := &model.PageInfo{
		HasNextPage:     hasNextPage,
		HasPreviousPage: after != nil && *after != "",
	}

	if len(edges) > 0 {
		start := edges[0].Cursor
		end := edges[len(edges)-1].Cursor
		pageInfo.StartCursor = &start
		pageInfo.EndCursor = &end
	}

	return &model.CensusValueConnection{
		Edges:    edges,
		PageInfo: pageInfo,
	}, nil
}

type censusSourceResolver struct{ *Resolver }

func (r *censusSourceResolver) Layers(ctx context.Context, obj *model.CensusSource) (ret []*model.CensusLayer, err error) {
	return LoaderFor(ctx).CensusSourceLayersBySourceIDs.Load(ctx, obj.ID)()
}

func (r *censusSourceResolver) Geographies(ctx context.Context, obj *model.CensusSource, limit *int, where *model.CensusSourceGeographyFilter) (ret []*model.CensusGeography, err error) {
	return LoaderFor(ctx).CensusGeographiesBySourceIDs.Load(ctx, censusSourceGeographyLoaderParam{SourceID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_CENSUS_MAXLIMIT), Where: where, Select: censusGeographySelect(ctx)})()
}

func (r *censusSourceResolver) Tables(ctx context.Context, obj *model.CensusSource, limit *int, where *model.CensusTableFilter) (ret []*model.CensusTable, err error) {
	return LoaderFor(ctx).CensusTablesBySourceIDs.Load(ctx, censusSourceTableLoaderParam{SourceID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_CENSUS_MAXLIMIT), Where: where})()
}

type censusGeographyResolver struct{ *Resolver }

func (r *censusGeographyResolver) Values(ctx context.Context, obj *model.CensusGeography, tableNames []string, datasetName *string, limit *int) ([]*model.CensusValue, error) {
	// dataloader cant easily pass []string
	return LoaderFor(ctx).CensusValuesByGeographyIDs.Load(ctx, censusValueLoaderParam{Dataset: datasetName, TableNames: strings.Join(tableNames, ","), Limit: resolverCheckLimitMax(limit, RESOLVER_CENSUS_MAXLIMIT), Geoid: *obj.Geoid})()
}

func (r *censusGeographyResolver) Layer(ctx context.Context, obj *model.CensusGeography) (*model.CensusLayer, error) {
	return LoaderFor(ctx).CensusLayersByIDs.Load(ctx, obj.LayerID)()
}

func (r *censusGeographyResolver) Source(ctx context.Context, obj *model.CensusGeography) (*model.CensusSource, error) {
	return LoaderFor(ctx).CensusSourcesByIDs.Load(ctx, obj.SourceID)()
}

type censusValueResolver struct{ *Resolver }

func (r *censusValueResolver) Table(ctx context.Context, obj *model.CensusValue) (*model.CensusTable, error) {
	return LoaderFor(ctx).CensusTableByIDs.Load(ctx, obj.TableID)()
}

type censusTableResolver struct{ *Resolver }

func (r *censusTableResolver) Fields(ctx context.Context, obj *model.CensusTable) ([]*model.CensusField, error) {
	return LoaderFor(ctx).CensusFieldsByTableIDs.Load(ctx, censusFieldLoaderParam{TableID: obj.ID})()
}

type censusLayerResolver struct{ *Resolver }

func (r *censusLayerResolver) Geographies(ctx context.Context, obj *model.CensusLayer, limit *int, where *model.CensusSourceGeographyFilter) (ret []*model.CensusGeography, err error) {
	return LoaderFor(ctx).CensusGeographiesByLayerIDs.Load(ctx, censusSourceGeographyLoaderParam{LayerID: obj.ID, Limit: resolverCheckLimitMax(limit, RESOLVER_CENSUS_MAXLIMIT), Where: where, Select: censusGeographySelect(ctx)})()
}

// add geography resolvers to agency, route, stop

func (r *agencyResolver) CensusGeographies(ctx context.Context, obj *model.Agency, limit *int, where *model.CensusGeographyFilter) ([]*model.CensusGeography, error) {
	return LoaderFor(ctx).CensusGeographiesByEntityIDs.Load(ctx, censusGeographyLoaderParam{
		EntityType: "agency",
		EntityID:   obj.ID,
		Limit:      resolverCheckLimitMax(limit, RESOLVER_CENSUS_MAXLIMIT),
		Where:      where,
		Select:     censusGeographySelect(ctx),
	})()
}

func (r *routeResolver) CensusGeographies(ctx context.Context, obj *model.Route, limit *int, where *model.CensusGeographyFilter) ([]*model.CensusGeography, error) {
	return LoaderFor(ctx).CensusGeographiesByEntityIDs.Load(ctx, censusGeographyLoaderParam{
		EntityType: "route",
		EntityID:   obj.ID,
		Limit:      resolverCheckLimitMax(limit, RESOLVER_CENSUS_MAXLIMIT),
		Where:      where,
		Select:     censusGeographySelect(ctx),
	})()
}

func (r *stopResolver) CensusGeographies(ctx context.Context, obj *model.Stop, limit *int, where *model.CensusGeographyFilter) ([]*model.CensusGeography, error) {
	return LoaderFor(ctx).CensusGeographiesByEntityIDs.Load(ctx, censusGeographyLoaderParam{
		EntityType: "stop",
		EntityID:   obj.ID,
		Limit:      resolverCheckLimitMax(limit, RESOLVER_CENSUS_MAXLIMIT),
		Where:      where,
		Select:     censusGeographySelect(ctx),
	})()
}

// censusGeographySelect reports which geometry columns the current census
// geography field selects.
func censusGeographySelect(ctx context.Context) model.CensusGeographySelect {
	return model.CensusGeographySelect{
		Geometry:             containsField(ctx, "geometry"),
		GeometryArea:         containsField(ctx, "geometry_area"),
		IntersectionArea:     containsField(ctx, "intersection_area"),
		IntersectionGeometry: containsField(ctx, "intersection_geometry"),
	}
}

// censusGeographiesByEntityIDs returns the census geographies within a radius
// of each stop, or of the stops of each route or agency. Each entity's stops
// become buffer points keyed by the entity's ID, which its matches report.
func censusGeographiesByEntityIDs(ctx context.Context, finder model.Finder, limit *int, where *model.CensusGeographyFilter, sel model.CensusGeographySelect, entityType string, ids []int) ([][]*model.CensusGeography, error) {
	if where == nil {
		where = &model.CensusGeographyFilter{}
	}
	search := func(points []*model.BufferPoint) ([]*model.CensusGeography, error) {
		return finder.FindCensusGeographies(ctx, limit, &model.CensusDatasetGeographyFilter{
			Dataset: where.Dataset,
			Layer:   where.Layer,
			Search:  where.Search,
			Location: &model.CensusDatasetGeographyLocationFilter{
				StopBuffer: &model.StopBuffer{Points: points, Radius: where.Radius},
			},
		}, sel)
	}
	var ents []*model.CensusGeography
	switch entityType {
	case "stop":
		// One search for every stop, each keyed by its own ID.
		stops, errs := finder.StopsByIDs(ctx, ids)
		if err := firstError(errs); err != nil {
			return nil, err
		}
		var points []*model.BufferPoint
		for _, stop := range stops {
			if p := stopBufferPoint(stop); p != nil {
				p.ID = &stop.ID
				points = append(points, p)
			}
		}
		// A search without points would have no buffer, and match everything.
		if len(points) > 0 {
			found, err := search(points)
			if err != nil {
				return nil, err
			}
			ents = found
		}
	case "route", "agency":
		// One search for each entity, over the union of its stops' buffers.
		for _, id := range ids {
			var groups [][]*model.Stop
			var err error
			if entityType == "route" {
				groups, err = finder.StopsByRouteIDs(ctx, nil, nil, []int{id})
			} else {
				groups, err = finder.StopsByAgencyIDs(ctx, nil, nil, nil, []int{id})
			}
			if err != nil {
				return nil, err
			}
			var points []*model.BufferPoint
			for _, stops := range groups {
				for _, stop := range stops {
					if p := stopBufferPoint(stop); p != nil {
						p.ID = &id
						points = append(points, p)
					}
				}
			}
			// An entity with no stops has no buffer, so it matches no
			// geographies.
			if len(points) == 0 {
				continue
			}
			found, err := search(points)
			if err != nil {
				return nil, err
			}
			ents = append(ents, found...)
		}
	default:
		return nil, fmt.Errorf("no census geographies for entity type %q", entityType)
	}
	byID := map[int][]*model.CensusGeography{}
	for _, ent := range ents {
		if ent.MatchID != nil {
			byID[*ent.MatchID] = append(byID[*ent.MatchID], ent)
		}
	}
	ret := make([][]*model.CensusGeography, len(ids))
	for i, id := range ids {
		ret[i] = byID[id]
	}
	return ret, nil
}

// datasetGeographyFilterWithStops returns the filter with its stop buffer's
// stops added to its points.
func datasetGeographyFilterWithStops(ctx context.Context, finder model.Finder, where *model.CensusDatasetGeographyFilter) (*model.CensusDatasetGeographyFilter, error) {
	if where == nil {
		return nil, nil
	}
	w := *where
	var err error
	w.Location, err = withStopPoints(ctx, finder, w.Location)
	return &w, err
}

// sourceGeographyFilterWithStops returns the filter with its stop buffer's
// stops added to its points.
func sourceGeographyFilterWithStops(ctx context.Context, finder model.Finder, where *model.CensusSourceGeographyFilter) (*model.CensusSourceGeographyFilter, error) {
	if where == nil {
		return nil, nil
	}
	w := *where
	var err error
	w.Location, err = withStopPoints(ctx, finder, w.Location)
	return &w, err
}

// withStopPoints returns the location filter with its stop buffer's stops added
// to its points, which are all that a finder's census search reads. The stops
// carry no key, so they share a buffer with the points that carry none.
func withStopPoints(ctx context.Context, finder model.Finder, loc *model.CensusDatasetGeographyLocationFilter) (*model.CensusDatasetGeographyLocationFilter, error) {
	if loc == nil || loc.StopBuffer == nil || len(loc.StopBuffer.StopIds) == 0 {
		return loc, nil
	}
	stops, errs := finder.StopsByIDs(ctx, loc.StopBuffer.StopIds)
	if err := firstError(errs); err != nil {
		return nil, err
	}
	buffer := *loc.StopBuffer
	buffer.Points = slices.Clone(buffer.Points)
	for _, stop := range stops {
		if p := stopBufferPoint(stop); p != nil {
			buffer.Points = append(buffer.Points, p)
		}
	}
	w := *loc
	w.StopBuffer = &buffer
	return &w, nil
}

// stopBufferPoint returns a stop's location as a buffer point without keys, or
// nil for a stop that has no location.
func stopBufferPoint(stop *model.Stop) *model.BufferPoint {
	if stop == nil || !stop.Geometry.Valid {
		return nil
	}
	return &model.BufferPoint{Lon: stop.Geometry.X(), Lat: stop.Geometry.Y()}
}

// firstError returns the first error of a lookup that returns one for each key.
func firstError(errs []error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
