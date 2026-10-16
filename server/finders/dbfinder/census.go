package dbfinder

import (
	"context"
	"errors"
	"math"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tlxy"
	sq "github.com/irees/squirrel"
	"github.com/twpayne/go-geom/encoding/geojson"
)

func (f *Finder) FindCensusDatasets(ctx context.Context, limit *int, after *model.Cursor, ids []int, where *model.CensusDatasetFilter) ([]*model.CensusDataset, error) {
	var ents []*model.CensusDataset
	q := censusDatasetSelect(limit, after, ids, where)
	if err := dbutil.Select(ctx, f.db, q, &ents); err != nil {
		return nil, queryErr(ctx, err)
	}
	return ents, nil
}

func (f *Finder) CensusTableByIDs(ctx context.Context, ids []int) ([]*model.CensusTable, []error) {
	var ents []*model.CensusTable
	err := dbutil.Select(ctx,
		f.db,
		quickSelect("tl_census_tables", nil, nil, ids),
		&ents,
	)
	if err != nil {
		return nil, queryErrs(ctx, len(ids), err)
	}
	return arrangeBy(ids, ents, func(ent *model.CensusTable) int { return ent.ID }), nil
}

func (f *Finder) CensusLayersByIDs(ctx context.Context, ids []int) ([]*model.CensusLayer, []error) {
	var ents []*model.CensusLayer
	err := dbutil.Select(ctx,
		f.db,
		quickSelect("tl_census_layers", nil, nil, ids),
		&ents,
	)
	if err != nil {
		return nil, queryErrs(ctx, len(ids), err)
	}
	return arrangeBy(ids, ents, func(ent *model.CensusLayer) int { return ent.ID }), nil
}

func (f *Finder) CensusSourcesByIDs(ctx context.Context, ids []int) ([]*model.CensusSource, []error) {
	var ents []*model.CensusSource
	err := dbutil.Select(ctx,
		f.db,
		quickSelect("tl_census_sources", nil, nil, ids),
		&ents,
	)
	if err != nil {
		return nil, queryErrs(ctx, len(ids), err)
	}
	return arrangeBy(ids, ents, func(ent *model.CensusSource) int { return ent.ID }), nil
}

func (f *Finder) CensusGeographiesByEntityIDs(ctx context.Context, limit *int, where *model.CensusGeographyFilter, sel model.CensusGeographySelect, entityType string, entityIds []int) ([][]*model.CensusGeography, error) {
	// Setup where clauses
	if where == nil {
		where = &model.CensusGeographyFilter{}
	}
	forStopids := func(stopIds []int) *model.CensusDatasetGeographyFilter {
		return &model.CensusDatasetGeographyFilter{
			Dataset: where.Dataset,
			Layer:   where.Layer,
			Search:  where.Search,
			Location: &model.CensusDatasetGeographyLocationFilter{
				StopBuffer: &model.StopBuffer{
					StopIds: stopIds,
					Radius:  where.Radius,
				},
			},
		}
	}

	var entityGeogs []*model.CensusGeography
	fields := censusGeographySelectFields{CensusGeographySelect: sel}
	if entityType == "stop" {
		// One batched query for every requested stop. The select keeps
		// per-stop attribution (match_entity_id = stop.id) so arrangeGroup
		// below can bucket results back to the right stop without needing a
		// loop-then-tag pass.
		fields.perStopAttribution = true
		var ents []*model.CensusGeography
		q, err := censusDatasetGeographySelect(limit, f.PermFilter(ctx), forStopids(entityIds), fields)
		if err != nil {
			return nil, err
		}
		if err := dbutil.Select(ctx, f.db, q, &ents); err != nil {
			return nil, queryErr(ctx, err)
		}
		entityGeogs = append(entityGeogs, ents...)
	} else {
		// Routes / agencies: the union over the entity's stop set is what we
		// want (avoids double-counting tracts hit by multiple stops). One
		// query per entity; tag MatchEntityID after scan since the SQL emits
		// 0 for the unioned buffer.
		for _, entityId := range entityIds {
			stopIds, err := getBufferStopIds(ctx, f.db, entityType, entityId)
			if err != nil {
				return nil, queryErr(ctx, err)
			}
			var ents []*model.CensusGeography
			q, err := censusDatasetGeographySelect(limit, f.PermFilter(ctx), forStopids(stopIds), fields)
			if err != nil {
				return nil, err
			}
			if err := dbutil.Select(ctx, f.db, q, &ents); err != nil {
				return nil, queryErr(ctx, err)
			}
			for _, ent := range ents {
				ent.MatchEntityID = entityId
			}
			entityGeogs = append(entityGeogs, ents...)
		}
	}
	ret := arrangeGroup(entityIds, entityGeogs, func(ent *model.CensusGeography) int { return ent.MatchEntityID })
	return ret, nil
}

func (f *Finder) CensusValuesByGeographyIDs(ctx context.Context, limit *int, datasetName string, tableNames []string, keys []string) ([][]*model.CensusValue, error) {
	var ents []*model.CensusValue
	err := dbutil.Select(
		ctx,
		f.db,
		censusValueSelect(limit, datasetName, tableNames, keys),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.CensusValue) string { return ent.Geoid }), queryErr(ctx, err)
}

func (f *Finder) CensusSourcesByDatasetIDs(ctx context.Context, limit *int, where *model.CensusSourceFilter, keys []int) ([][]*model.CensusSource, error) {
	q := censusSourceSelect(limit, nil, nil, where)
	var ents []*model.CensusSource
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			q,
			"tl_census_datasets",
			"id",
			"tl_census_sources",
			"dataset_id",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.CensusSource) int { return ent.DatasetID }), queryErr(ctx, err)
}

func (f *Finder) CensusDatasetLayersByDatasetIDs(ctx context.Context, keys []int) ([][]*model.CensusLayer, []error) {
	var ents []*model.CensusLayer
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			sq.StatementBuilder.Select("*").From("tl_census_layers"),
			"tl_census_datasets",
			"id",
			"tl_census_layers",
			"dataset_id",
			keys,
		),
		&ents,
	)
	if err != nil {
		return nil, queryErrs(ctx, len(keys), err)
	}
	return arrangeGroup(keys, ents, func(ent *model.CensusLayer) int { return ent.DatasetID }), nil
}

func (f *Finder) CensusSourceLayersBySourceIDs(ctx context.Context, keys []int) ([][]*model.CensusLayer, []error) {
	type qent struct {
		SourceID int
		model.CensusLayer
	}
	var ents []*qent
	// Dedup per (source, layer): the dataloader batches multiple source ids into one
	// query, and sources can share a layer, so deduping on layer id alone would drop a
	// shared layer from every source but one.
	q := sq.StatementBuilder.
		Select("tlcg.source_id", "tlcl.*").
		Distinct().Options("on (tlcg.source_id, tlcl.id)").
		From("tl_census_geographies tlcg").
		Join("tl_census_layers tlcl on tlcl.id = tlcg.layer_id").
		Where(sq.Eq{"tlcg.source_id": keys})
	err := dbutil.Select(ctx,
		f.db,
		q,
		&ents,
	)
	if err != nil {
		return nil, queryErrs(ctx, len(keys), err)
	}
	grouped := arrangeGroup(keys, ents, func(ent *qent) int { return ent.SourceID })
	var ret [][]*model.CensusLayer
	for _, group := range grouped {
		var g []*model.CensusLayer
		for _, ent := range group {
			g = append(g, &ent.CensusLayer)
		}
		ret = append(ret, g)
	}
	return ret, nil
}

func (f *Finder) CensusTablesBySourceIDs(ctx context.Context, limit *int, where *model.CensusTableFilter, keys []int) ([][]*model.CensusTable, error) {
	type qent struct {
		SourceID int
		model.CensusTable
	}
	var ents []*qent
	// Tables have no source_id; the source->table association lives in the loaded
	// values. Dedup per (source, table) since the loader batches multiple source ids
	// and a table can appear across sources.
	q := sq.StatementBuilder.
		Select("v.source_id", "tlct.*").
		Distinct().Options("on (v.source_id, tlct.id)").
		From("tl_census_values v").
		Join("tl_census_tables tlct on tlct.id = v.table_id").
		Where(sq.Eq{"v.source_id": keys}).
		OrderBy("v.source_id", "tlct.id")
	if where != nil && where.Search != nil {
		q = q.Where(sq.ILike{"tlct.table_name": dbutil.EscapeLike(*where.Search, true, true)})
	}
	if err := dbutil.Select(ctx, f.db, q, &ents); err != nil {
		return nil, queryErr(ctx, err)
	}
	grouped := arrangeGroup(keys, ents, func(ent *qent) int { return ent.SourceID })
	var ret [][]*model.CensusTable
	for _, group := range grouped {
		var g []*model.CensusTable
		for _, ent := range group {
			g = append(g, &ent.CensusTable)
		}
		if limit != nil && *limit >= 0 && len(g) > *limit {
			g = g[:*limit]
		}
		ret = append(ret, g)
	}
	return ret, nil
}

func (f *Finder) CensusGeographiesByDatasetIDs(ctx context.Context, limit *int, p *model.CensusDatasetGeographyFilter, sel model.CensusGeographySelect, keys []int) ([][]*model.CensusGeography, error) {
	var ents []*model.CensusGeography
	q, err := censusDatasetGeographySelect(limit, f.PermFilter(ctx), p, censusGeographySelectFields{CensusGeographySelect: sel})
	if err != nil {
		return nil, err
	}
	err = dbutil.Select(ctx,
		f.db,
		q.Where(sq.Eq{"tlcd.id": keys}),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.CensusGeography) int { return ent.DatasetID }), queryErr(ctx, err)
}

func (f *Finder) CensusGeographiesByLayerIDs(ctx context.Context, limit *int, where *model.CensusSourceGeographyFilter, sel model.CensusGeographySelect, keys []int) ([][]*model.CensusGeography, error) {
	w := &model.CensusDatasetGeographyFilter{}
	if where != nil {
		w.Ids = where.Ids
		w.Search = where.Search
		w.Location = where.Location
	}
	var ents []*model.CensusGeography
	q, err := censusDatasetGeographySelect(limit, f.PermFilter(ctx), w, censusGeographySelectFields{CensusGeographySelect: sel})
	if err != nil {
		return nil, err
	}
	err = dbutil.Select(ctx,
		f.db,
		lateralWrap(
			q,
			"tl_census_layers",
			"id",
			"tlcg",
			"layer_id",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.CensusGeography) int { return ent.LayerID }), queryErr(ctx, err)
}

func (f *Finder) CensusGeographiesBySourceIDs(ctx context.Context, limit *int, where *model.CensusSourceGeographyFilter, sel model.CensusGeographySelect, keys []int) ([][]*model.CensusGeography, error) {
	w := &model.CensusDatasetGeographyFilter{}
	if where != nil {
		w.Ids = where.Ids
		w.Search = where.Search
		w.Location = where.Location
	}
	var ents []*model.CensusGeography
	q, err := censusDatasetGeographySelect(limit, f.PermFilter(ctx), w, censusGeographySelectFields{CensusGeographySelect: sel})
	if err != nil {
		return nil, err
	}
	err = dbutil.Select(ctx,
		f.db,
		lateralWrap(
			q,
			"tl_census_sources",
			"id",
			"tlcg",
			"source_id",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.CensusGeography) int { return ent.SourceID }), queryErr(ctx, err)
}

func (f *Finder) CensusFieldsByTableIDs(ctx context.Context, limit *int, keys []int) ([][]*model.CensusField, error) {
	var ents []*model.CensusField
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			quickSelectOrder("tl_census_fields", limit, nil, nil, "id"),
			"tl_census_tables",
			"id",
			"tl_census_fields",
			"table_id",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.CensusField) int { return ent.TableID }), queryErr(ctx, err)
}

func (f *Finder) CensusTablesByDatasetIDs(ctx context.Context, limit *int, where *model.CensusTableFilter, keys []int) ([][]*model.CensusTable, error) {
	q := censusTableSelect(limit, where)
	var ents []*model.CensusTable
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			q,
			"tl_census_datasets",
			"id",
			"tl_census_tables",
			"dataset_id",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.CensusTable) int { return ent.DatasetID }), queryErr(ctx, err)
}

// FindCensusValuesByDatasetID finds census values by dataset ID with optional filters.
// Supports Relay-style cursor pagination for large result sets (e.g., querying all NTD agencies).
// Cursor encodes composite key (geoid, table_id) to enable proper pagination.
func (f *Finder) FindCensusValuesByDatasetID(ctx context.Context, limit *int, after model.CensusCursor, datasetID int, where *model.CensusDatasetValueFilter) ([]*model.CensusValue, error) {
	var ents []*model.CensusValue
	q := censusValueFilterSelect(limit, after, datasetID, where)
	if err := dbutil.Select(ctx, f.db, q, &ents); err != nil {
		return nil, queryErr(ctx, err)
	}
	return ents, nil
}

func censusDatasetSelect(_ *int, _ *model.Cursor, _ []int, where *model.CensusDatasetFilter) sq.SelectBuilder {
	q := sq.StatementBuilder.
		Select("*").
		From("tl_census_datasets")
	if where != nil {
		if where.Name != nil {
			q = q.Where(sq.Eq{"name": *where.Name})
		}
		if where.Search != nil {
			q = q.Where(sq.Like{"name": dbutil.EscapeLike(*where.Search, true, true)})
		}
	}
	return q
}

func censusSourceSelect(limit *int, after *model.Cursor, ids []int, where *model.CensusSourceFilter) sq.SelectBuilder {
	q := quickSelectOrder("tl_census_sources", limit, after, ids, "id")
	if where != nil {
		if where.Name != nil {
			q = q.Where(sq.Eq{"name": *where.Name})
		}
	}
	return q
}

type censusGeographySelectFields struct {
	model.CensusGeographySelect
	// Caller flag (not GraphQL-driven): when true, the buffer CTE for the
	// `stop_buffer` filter emits one row per stop with match_entity_id =
	// gtfs_stops.id instead of unioning all stops into a single polygon.
	// Used by CensusGeographiesByEntityIDs for entityType == "stop" so a
	// single batched query can be grouped back per requesting stop.
	perStopAttribution bool
}

// censusClipMaxVertices is the most vertices in a piece of a subdivided search
// area, ST_Subdivide's default. A `within` polygon this small is intersected
// whole.
const censusClipMaxVertices = 256

// censusSegmentDegrees is the longest edge, in degrees, kept by census area
// measurements and search areas.
//
// An edge is a straight line in lon/lat, which bends in the equal-area
// projection areas are measured in, and which the geography index reads as a
// great circle. Edges this short keep both within centimeters of the line.
const censusSegmentDegrees = 0.01

// errCensusAntimeridian reports a census search area that crosses the
// antimeridian, which planar intersections can't represent.
var errCensusAntimeridian = errors.New("census geography search areas that cross the antimeridian are not supported")

// errCensusStopBufferExclude reports a stop_buffer.exclude without both a
// query area and a positive radius.
var errCensusStopBufferExclude = errors.New("census geography stop_buffer.exclude requires a query area (bbox, within or geography_ids) and a positive radius")

func censusDatasetGeographySelect(limit *int, permFilter *model.PermFilter, where *model.CensusDatasetGeographyFilter, fields censusGeographySelectFields) (sq.SelectBuilder, error) {
	// Include matched entity column
	cols := []string{
		"tlcg.id",
		"tlcl.name as layer_name",
		"tlcg.geoid",
		"tlcg.name",
		"tlcg.aland",
		"tlcg.awater",
		"tlcg.adm0_name",
		"tlcg.adm1_name",
		"tlcg.adm0_iso",
		"tlcg.adm1_iso",
		"tlcs.name as source_name",
		"tlcs.id as source_id",
		"tlcd.name as dataset_name",
		"tlcd.id as dataset_id",
		"tlcg.layer_id as layer_id",
	}
	if fields.Geometry {
		cols = append(cols, "tlcg.geometry as geometry")
	}

	orderBy := sq.Expr("tlcg.id")

	// A normal query..
	q := sq.StatementBuilder.
		Select(cols...).
		From("tl_census_geographies tlcg").
		Join("tl_census_layers tlcl on tlcl.id = tlcg.layer_id").
		Join("tl_census_datasets tlcd on tlcd.id = tlcl.dataset_id").
		Join("tl_census_sources tlcs on tlcs.id = tlcg.source_id").
		Limit(finderCheckLimit(limit))
	if fields.GeometryArea {
		// Measured like intersection_area, so a geography inside the search area
		// reports the same value for both.
		q = q.Column("ST_Area(ST_Transform(ST_Segmentize(tlcg.geometry::geometry, ?), 6933)) as geometry_area", censusSegmentDegrees)
	}

	if where != nil && where.Location != nil {
		qBufferUse := false
		var qBuffer sq.SelectBuilder
		qPointsUse := false
		var qPoints sq.SelectBuilder
		// Set for search areas that can be too large to intersect whole: a stop
		// union, a union of geographies, and a `within` polygon past
		// censusClipMaxVertices.
		subdivide := false

		loc := where.Location
		withinClip := loc.Within != nil && loc.Within.Valid
		geographyIdsClip := len(loc.GeographyIds) > 0
		stopBufferClip := loc.StopBuffer != nil && len(loc.StopBuffer.StopIds) > 0
		stopBufferRadius := 0.0
		stopBufferExclude := false
		// Read even without stop ids, so the exclude check below also covers an
		// empty stop list.
		if loc.StopBuffer != nil {
			// Clamp ceiling of 1600m ~ 1 mile, a common US analysis radius.
			stopBufferRadius = checkFloat(loc.StopBuffer.Radius, 0, 1_600)
			stopBufferExclude = loc.StopBuffer.Exclude != nil && *loc.StopBuffer.Exclude
		}
		if err := checkCensusAntimeridian(loc, stopBufferClip); err != nil {
			return q, err
		}

		// A request names the stops to buffer by id, read by primary key into a
		// materialized CTE. Every read of the stops in this select starts there.
		//
		// Gating gtfs_stops directly let the planner pick a plan that took minutes
		// for tens of thousands of ids, against seconds when the ids are read first.
		if stopBufferClip {
			q = q.WithCTE(sq.CTE{
				Alias:        "requested_stops",
				Materialized: true,
				Expression: sq.StatementBuilder.
					Select("gtfs_stops.id", "gtfs_stops.feed_version_id", "gtfs_stops.geometry").
					From("gtfs_stops").
					Where(In("gtfs_stops.id", loc.StopBuffer.StopIds)),
			})
		}
		stopsSelect := func() sq.SelectBuilder {
			return gatedRows("requested_stops", permFilter)
		}

		// Single-row clip-geometry builders (column `buffer`), shared between the
		// standalone branches and the combined query-area + stop_buffer case so
		// the geometry SQL lives in one place.
		bboxClipSelect := func() sq.SelectBuilder {
			return sq.StatementBuilder.Select().
				Column("ST_MakeEnvelope(?,?,?,?,4326) as buffer", loc.Bbox.MinLon, loc.Bbox.MinLat, loc.Bbox.MaxLon, loc.Bbox.MaxLat)
		}
		withinClipSelect := func() sq.SelectBuilder {
			// Repair a self-intersecting polygon, which would otherwise fail or
			// invert the search, then densify its long edges.
			jj, _ := geojson.Marshal(loc.Within.Val)
			return sq.StatementBuilder.Select().
				Column("ST_Segmentize(ST_CollectionExtract(ST_MakeValid(ST_GeomFromGeoJSON(?)), 3), ?) as buffer", string(jj), censusSegmentDegrees)
		}
		geographyIdsClipSelect := func() sq.SelectBuilder {
			// No matching ids leaves a NULL union, which matches nothing.
			return sq.StatementBuilder.Select().
				Column("ST_Union(clip_geographies.geometry::geometry) as buffer").
				From("tl_census_geographies clip_geographies").
				Where(sq.Eq{"clip_geographies.id": loc.GeographyIds})
		}
		stopUnionClipSelect := func() sq.SelectBuilder {
			return stopsSelect().
				Column("ST_Union(ST_Buffer(requested_stops.geometry::geography, ?)::geometry) as buffer", stopBufferRadius)
		}

		// The query area, however it was expressed: the first of `bbox`,
		// `within` and `geography_ids` that is set.
		var areaClipSelect func() sq.SelectBuilder
		switch {
		case loc.Bbox != nil:
			areaClipSelect = bboxClipSelect
		case withinClip:
			areaClipSelect = withinClipSelect
		case geographyIdsClip:
			areaClipSelect = geographyIdsClipSelect
		}
		areaClip := areaClipSelect != nil

		// Exclude subtracts the buffers from a query area, and radius 0 already
		// means matching the geographies that contain a stop.
		if stopBufferExclude && (!areaClip || stopBufferRadius == 0) {
			return q, errCensusStopBufferExclude
		}

		// Filter precedence: only the query area and `stop_buffer` compose.
		//
		// With a positive radius, the search area is the part of the query area
		// inside the stop buffers, or with exclude, the part outside them. With
		// radius 0, a geography matches when it contains a stop inside the query
		// area. Otherwise the first filter present in this chain applies and the
		// rest are ignored. In per-stop attribution mode (the internal by-entity
		// loaders), a positive-radius stop_buffer takes precedence and the query
		// area is not applied.
		if areaClip && stopBufferClip && stopBufferRadius > 0 && !fields.perStopAttribution {
			// Clip the query area to the stop-buffer union: their intersection, or
			// with exclude, their difference. Without this case the chain below
			// would honor only the buffer and ignore the query area.
			qBufferUse = true
			subdivide = true
			q = q.WithCTE(sq.CTE{Alias: "clip_area", Materialized: true, Expression: areaClipSelect()})
			q = q.WithCTE(sq.CTE{Alias: "clip_stops", Materialized: true, Expression: stopUnionClipSelect()})
			combine := "ST_Intersection(clip_area.buffer, clip_stops.buffer) as buffer"
			if stopBufferExclude {
				// The union is NULL when no requested stop is visible. Default it to
				// empty, so hidden stops subtract nothing rather than nulling the
				// difference.
				combine = "ST_Difference(clip_area.buffer, COALESCE(clip_stops.buffer, 'SRID=4326;GEOMETRYCOLLECTION EMPTY'::geometry)) as buffer"
			}
			qBuffer = sq.StatementBuilder.Select().
				Column(combine).
				Column("0 as match_entity_id").
				From("clip_area").
				JoinClause("CROSS JOIN clip_stops")
		} else if loc.Bbox != nil && !stopBufferClip {
			qBufferUse = true
			qBuffer = bboxClipSelect().Column("0 as match_entity_id")
		} else if withinClip && !stopBufferClip {
			qBufferUse = true
			subdivide = len(loc.Within.Val.FlatCoords())/loc.Within.Val.Stride() > censusClipMaxVertices
			qBuffer = withinClipSelect().Column("0 as match_entity_id")
		} else if geographyIdsClip && !stopBufferClip {
			// Always cut: the union's vertices can't be counted before the
			// query runs, and an admin boundary often has thousands.
			qBufferUse = true
			subdivide = true
			qBuffer = geographyIdsClipSelect().Column("0 as match_entity_id")
		} else if loc.Near != nil && !stopBufferClip {
			// Guarded like the branches above so a stop_buffer isn't discarded
			// here: without it a radius-0 buffer alongside `near` would skip
			// the point-matching branch entirely.
			radius := checkFloat(&loc.Near.Radius, 0, 1_000_000)
			qBufferUse = true
			qBuffer = sq.StatementBuilder.Select().
				Column("ST_Buffer(ST_MakePoint(?,?)::geography, ?)::geometry as buffer", loc.Near.Lon, loc.Near.Lat, radius).
				Column("0 as match_entity_id")
		} else if stopBufferClip {
			if stopBufferRadius == 0 {
				// Per-stop attribution keeps a row per stop. Otherwise every stop
				// shares one id, so the GROUP BY below lists a geography once.
				qPointsUse = true
				qPoints = stopsSelect().Column("requested_stops.geometry as buffer")
				if fields.perStopAttribution {
					qPoints = qPoints.Column("requested_stops.id as match_entity_id")
				} else {
					qPoints = qPoints.Column("0 as match_entity_id")
				}
				if areaClip {
					// A combined query area restricts point matching to stops
					// inside it rather than being silently ignored.
					q = q.WithCTE(sq.CTE{Alias: "clip_area", Materialized: true, Expression: areaClipSelect()})
					qPoints = qPoints.
						JoinClause("CROSS JOIN clip_area").
						Where(sq.Expr("ST_Intersects(clip_area.buffer, requested_stops.geometry)"))
				}
			} else if fields.perStopAttribution {
				// One buffer per stop, attribution preserved as
				// match_entity_id. Tract rows are duplicated when a tract
				// intersects multiple stops in the input set; that's the
				// intent — callers want per-stop apportionment.
				qBufferUse = true
				qBuffer = stopsSelect().
					Column("requested_stops.id as match_entity_id").
					Column("ST_Buffer(requested_stops.geometry::geography, ?)::geometry as buffer", stopBufferRadius)
			} else {
				// Default: union over the input stop set. Used by
				// routes/agencies (which want the union over their stops) and
				// by top-level aggregation queries.
				qBufferUse = true
				subdivide = true
				qBuffer = stopUnionClipSelect().Column("0 as match_entity_id")
			}
		}
		if qBufferUse {
			if subdivide {
				// Cut the search area into small pieces, so each geography is
				// intersected with just the union of the pieces it touches.
				//
				// Per-stop buffers, and a bbox or near circle on its own, are
				// never cut. ST_Subdivide's planner estimate is 1,000 rows per
				// input row, which multiplied across per-stop buffers or batched
				// loaders makes Postgres JIT-compile otherwise fast queries.
				qBuffer = sq.StatementBuilder.
					Select("clip.match_entity_id").
					Column("ST_Segmentize(ST_Subdivide(ST_CollectionExtract(clip.buffer, 3), ?), ?) as buffer", censusClipMaxVertices, censusSegmentDegrees).
					FromSelect(qBuffer, "clip")
			}
			q = q.WithCTE(sq.CTE{
				Alias:        "buffer",
				Materialized: true,
				Expression:   qBuffer,
			})
			q = q.Join("buffer ON tlcg.geometry && buffer.buffer").
				Where(sq.Expr("ST_Intersects(tlcg.geometry::geometry, buffer.buffer)"))
			// Intersect in lon/lat, then measure the result like geometry_area:
			// densified, in the equal-area projection EPSG:6933.
			//
			// Geography functions would reproject the search area on every row,
			// and would read a long edge along a parallel, such as a bbox side,
			// as a great circle.
			if subdivide {
				q = q.GroupBy("tlcg.id", "tlcl.id", "tlcd.id", "tlcs.id", "buffer.match_entity_id")
				if fields.IntersectionArea {
					q = q.Column("ST_Area(ST_Transform(ST_Segmentize(ST_Intersection(tlcg.geometry::geometry, ST_Union(buffer.buffer)), ?), 6933)) as intersection_area", censusSegmentDegrees)
				}
				if fields.IntersectionGeometry {
					q = q.Column("ST_Intersection(tlcg.geometry::geometry, ST_Union(buffer.buffer)) as intersection_geometry")
				}
			} else {
				if fields.IntersectionArea {
					q = q.Column("ST_Area(ST_Transform(ST_Segmentize(ST_Intersection(tlcg.geometry::geometry, buffer.buffer), ?), 6933)) as intersection_area", censusSegmentDegrees)
				}
				if fields.IntersectionGeometry {
					q = q.Column("ST_Intersection(tlcg.geometry::geometry, buffer.buffer) as intersection_geometry")
				}
			}
			if fields.perStopAttribution {
				q = q.Column("buffer.match_entity_id")
			}
		}
		if qPointsUse {
			q = q.WithCTE(sq.CTE{
				Alias:        "buffer",
				Materialized: true,
				Expression:   qPoints,
			})
			q = q.Join("buffer ON tlcg.geometry && buffer.buffer").
				Column("buffer.match_entity_id").
				Where(sq.Expr("ST_Intersects(tlcg.geometry::geometry, buffer.buffer)")).
				GroupBy("tlcg.id", "tlcl.id", "tlcd.id", "tlcs.id", "buffer.match_entity_id")
		}
		// Pin layer_id as a scalar so the composite (layer_id, geometry) GiST
		// index can serve the spatial join. Only safe when both dataset and
		// layer are set, since (dataset, layer) resolves to a single layer_id.
		if (qBufferUse || qPointsUse) && where.Dataset != nil && where.Layer != nil {
			layerID := sq.StatementBuilder.
				Select("clsub.id").
				From("tl_census_layers clsub").
				Join("tl_census_datasets cdsub ON cdsub.id = clsub.dataset_id").
				Where(sq.Eq{"clsub.name": *where.Layer, "cdsub.name": *where.Dataset})
			q = q.Where(layerID.Prefix("tlcg.layer_id = (").Suffix(")"))
		}
		if loc.Focus != nil {
			orderBy = sq.Expr("ST_Distance(tlcg.geometry, ST_MakePoint(?,?))", loc.Focus.Lon, loc.Focus.Lat)
		}
	}

	// Check layer, dataset
	if where != nil {
		if where.Dataset != nil {
			q = q.Where(sq.Eq{"tlcd.name": *where.Dataset})
		}
		if where.Layer != nil {
			q = q.Where(sq.Eq{"tlcl.name": where.Layer})
		}
		if where.Search != nil {
			q = q.Where(sq.ILike{"tlcg.name": dbutil.EscapeLike(*where.Search, true, true)})
		}
		if len(where.Ids) > 0 {
			q = q.Where(sq.Eq{"tlcg.id": where.Ids})
		}
	}

	q = q.OrderByClause(orderBy)
	return q, nil
}

// checkCensusAntimeridian returns errCensusAntimeridian when the search area
// that takes effect crosses the antimeridian. Stop buffers aren't checked,
// since a stop would have to lie within 1,600 m of the antimeridian.
func checkCensusAntimeridian(loc *model.CensusDatasetGeographyLocationFilter, stopBufferClip bool) error {
	switch {
	case loc.Bbox != nil:
		b := loc.Bbox
		if b.MinLon < -180 || b.MaxLon > 180 || b.MinLon > b.MaxLon {
			return errCensusAntimeridian
		}
	case loc.Within != nil && loc.Within.Valid:
		// A polygon wider than 180° of longitude was drawn across the
		// antimeridian with wrapped coordinates.
		b := loc.Within.Val.Bounds()
		if b.Min(0) < -180 || b.Max(0) > 180 || b.Max(0)-b.Min(0) > 180 {
			return errCensusAntimeridian
		}
	case len(loc.GeographyIds) > 0:
		// The named geographies are stored like the ones they clip, so even
		// one split at the antimeridian clips correctly.
		return nil
	case loc.Near != nil && !stopBufferClip:
		// The circle's half width in longitude, on a sphere. A circle that
		// reaches a pole spans every longitude.
		d := checkFloat(&loc.Near.Radius, 0, 1_000_000) / tlxy.EarthRadiusMeters
		lat := loc.Near.Lat * math.Pi / 180
		if math.Abs(lat)+d >= math.Pi/2 {
			return errCensusAntimeridian
		}
		halfWidth := math.Asin(math.Sin(d)/math.Cos(lat)) * 180 / math.Pi
		if math.Abs(loc.Near.Lon)+halfWidth > 180 {
			return errCensusAntimeridian
		}
	}
	return nil
}

func getBufferStopIds(ctx context.Context, db tldb.Ext, entityType string, entityId int) ([]int, error) {
	// Handle aggregation by entity type
	q := sq.StatementBuilder.
		Select("id").
		Distinct().Options("on (gtfs_stops.id)").
		From("gtfs_stops")
	switch entityType {
	case "route":
		q = q.Join("tl_route_stops ON tl_route_stops.stop_id = gtfs_stops.id").Where(sq.Eq{"tl_route_stops.route_id": entityId})
	case "agency":
		q = q.Join("tl_route_stops ON tl_route_stops.stop_id = gtfs_stops.id").Where(sq.Eq{"tl_route_stops.agency_id": entityId})
	case "stop":
		// No need to query, just return the single stop ID
		return []int{entityId}, nil
	}
	var stopIds []int
	err := dbutil.Select(ctx, db, q, &stopIds)
	return stopIds, err
}

func censusValueSelect(limit *int, datasetName string, tnames []string, geoids []string) sq.SelectBuilder {
	q := sq.StatementBuilder.
		Select(
			"tlcv.table_values as values",
			"tlcv.geoid",
			"tlcv.table_id",
			"tlcs.name as source_name",
			"tlcd.name as dataset_name",
		).
		From("tl_census_values tlcv").
		Limit(finderCheckLimit(limit)).
		Join("tl_census_tables tlct ON tlct.id = tlcv.table_id").
		Join("tl_census_sources tlcs on tlcs.id = tlcv.source_id").
		Join("tl_census_datasets tlcd on tlcd.id = tlct.dataset_id").
		Where(sq.Eq{"tlcv.geoid": geoids}).
		Where(sq.Eq{"tlct.table_name": tnames}).
		OrderBy("tlcv.table_id")
	if datasetName != "" {
		q = q.Where(sq.Eq{"tlcd.name": datasetName})
	}
	return q
}

func censusTableSelect(limit *int, where *model.CensusTableFilter) sq.SelectBuilder {
	q := quickSelectOrder("tl_census_tables", limit, nil, nil, "id")
	if where != nil {
		if where.Search != nil {
			q = q.Where(sq.ILike{"table_name": dbutil.EscapeLike(*where.Search, true, true)})
		}
	}
	return q
}

func censusValueFilterSelect(limit *int, after model.CensusCursor, datasetID int, where *model.CensusDatasetValueFilter) sq.SelectBuilder {
	q := sq.StatementBuilder.
		Select(
			"tlcv.table_values as values",
			"tlcv.geoid",
			"tlcv.table_id",
			"tlcs.name as source_name",
			"tlcd.name as dataset_name",
		).
		From("tl_census_values tlcv").
		Limit(finderCheckLimit(limit)).
		Join("tl_census_tables tlct ON tlct.id = tlcv.table_id").
		Join("tl_census_sources tlcs on tlcs.id = tlcv.source_id").
		Join("tl_census_datasets tlcd on tlcd.id = tlct.dataset_id").
		Where(sq.Eq{"tlcd.id": datasetID}).
		OrderBy("tlcv.geoid", "tlcv.table_id")

	// Composite key cursor pagination using (geoid, table_id)
	// Implements: WHERE (geoid, table_id) > (last_geoid, last_table_id)
	// SQL: WHERE geoid > last_geoid OR (geoid = last_geoid AND table_id > last_table_id)
	if after.Valid {
		q = q.Where(sq.Or{
			sq.Gt{"tlcv.geoid": after.Geoid},
			sq.And{
				sq.Eq{"tlcv.geoid": after.Geoid},
				sq.Gt{"tlcv.table_id": after.TableID},
			},
		})
	}

	if where != nil {
		if where.Table != nil {
			q = q.Where(sq.Eq{"tlct.table_name": *where.Table})
		}
		if where.Geoid != nil {
			q = q.Where(sq.Eq{"tlcv.geoid": *where.Geoid})
		}
		if where.GeoidPrefix != nil {
			q = q.Where(sq.Like{"tlcv.geoid": dbutil.EscapeLike(*where.GeoidPrefix, false, true)})
		}
	}
	return q
}
