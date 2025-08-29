package dbfinder

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tldb"
	sq "github.com/irees/squirrel"
	"github.com/twpayne/go-geom/encoding/geojson"
)

func (f *Finder) FindCensusDatasets(ctx context.Context, limit *int, after *model.Cursor, ids []int, where *model.CensusDatasetFilter) ([]*model.CensusDataset, error) {
	var ents []*model.CensusDataset
	q := censusDatasetSelect(limit, after, ids, where)
	if err := dbutil.Select(ctx, f.db, q, &ents); err != nil {
		return nil, logErr(ctx, err)
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
		return nil, logExtendErr(ctx, len(ids), err)
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
		return nil, logExtendErr(ctx, len(ids), err)
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
		return nil, logExtendErr(ctx, len(ids), err)
	}
	return arrangeBy(ids, ents, func(ent *model.CensusSource) int { return ent.ID }), nil
}

func (f *Finder) CensusGeographiesByEntityIDs(ctx context.Context, limit *int, where *model.CensusGeographyFilter, entityType string, entityIds []int) ([][]*model.CensusGeography, error) {
	// Sadly cannot be optimized to avoid N+1
	var ret [][]*model.CensusGeography
	fields := getCensusGeographySelectFields(ctx)
	for _, entityId := range entityIds {
		if where == nil {
			where = &model.CensusGeographyFilter{}
		}
		stopIds, err := getBufferStopIds(ctx, f.db, entityType, entityId)
		if err != nil {
			return nil, logErr(ctx, err)
		}
		var ents []*model.CensusGeography
		pw := &model.CensusDatasetGeographyFilter{
			Layer:  where.Layer,
			Search: where.Search,
			Location: &model.CensusDatasetGeographyLocationFilter{
				StopBuffer: &model.StopBuffer{
					StopIds: stopIds,
					Radius:  where.Radius,
				},
			},
		}
		if err := dbutil.Select(ctx, f.db, censusDatasetGeographySelect(limit, pw, fields), &ents); err != nil {
			return nil, logErr(ctx, err)
		}
		ret = append(ret, ents)
	}
	return ret, nil
}

func (f *Finder) CensusValuesByGeographyIDs(ctx context.Context, limit *int, tableNames []string, keys []string) ([][]*model.CensusValue, error) {
	var ents []*model.CensusValue
	err := dbutil.Select(
		ctx,
		f.db,
		censusValueSelect(limit, "", tableNames, keys),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.CensusValue) string { return ent.Geoid }), err
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
	return arrangeGroup(keys, ents, func(ent *model.CensusSource) int { return ent.DatasetID }), err
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
		return nil, logExtendErr(ctx, len(keys), err)
	}
	return arrangeGroup(keys, ents, func(ent *model.CensusLayer) int { return ent.DatasetID }), nil
}

func (f *Finder) CensusSourceLayersBySourceIDs(ctx context.Context, keys []int) ([][]*model.CensusLayer, []error) {
	type qent struct {
		SourceID int
		model.CensusLayer
	}
	var ents []*qent
	q := sq.StatementBuilder.
		Select("tlcg.source_id", "tlcl.*").
		Distinct().Options("on (tlcl.id)").
		From("tl_census_geographies tlcg").
		Join("tl_census_layers tlcl on tlcl.id = tlcg.layer_id").
		Where(sq.Eq{"tlcg.source_id": keys})
	err := dbutil.Select(ctx,
		f.db,
		q,
		&ents,
	)
	if err != nil {
		return nil, logExtendErr(ctx, len(keys), err)
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

func (f *Finder) CensusGeographiesByDatasetIDs(ctx context.Context, limit *int, p *model.CensusDatasetGeographyFilter, keys []int) ([][]*model.CensusGeography, error) {
	var ents []*model.CensusGeography
	q := censusDatasetGeographySelect(limit, p, getCensusGeographySelectFields(ctx))
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			q,
			"tl_census_datasets",
			"id",
			"tlcd",
			"id",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.CensusGeography) int { return ent.DatasetID }), err
}

func (f *Finder) CensusGeographiesByLayerIDs(ctx context.Context, limit *int, where *model.CensusSourceGeographyFilter, keys []int) ([][]*model.CensusGeography, error) {
	w := &model.CensusDatasetGeographyFilter{}
	if where != nil {
		w.Ids = where.Ids
		w.Search = where.Search
		w.Location = where.Location
	}
	var ents []*model.CensusGeography
	q := censusDatasetGeographySelect(limit, w, getCensusGeographySelectFields(ctx))
	err := dbutil.Select(ctx,
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
	return arrangeGroup(keys, ents, func(ent *model.CensusGeography) int { return ent.LayerID }), err
}

func (f *Finder) CensusGeographiesBySourceIDs(ctx context.Context, limit *int, where *model.CensusSourceGeographyFilter, keys []int) ([][]*model.CensusGeography, error) {
	w := &model.CensusDatasetGeographyFilter{}
	if where != nil {
		w.Ids = where.Ids
		w.Search = where.Search
		w.Location = where.Location
	}
	var ents []*model.CensusGeography
	q := censusDatasetGeographySelect(limit, w, getCensusGeographySelectFields(ctx))
	err := dbutil.Select(ctx,
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
	return arrangeGroup(keys, ents, func(ent *model.CensusGeography) int { return ent.SourceID }), err
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
	return arrangeGroup(keys, ents, func(ent *model.CensusField) int { return ent.TableID }), err
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
			q = q.Where(sq.Like{"name": fmt.Sprintf("%%%s%%", *where.Search)})
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
	intersectionArea     bool
	intersectionGeometry bool
	geometryArea         bool
	geometry             bool
}

func getCensusGeographySelectFields(ctx context.Context) censusGeographySelectFields {
	fields := censusGeographySelectFields{}
	if containsField(ctx, "geometry") {
		fields.geometry = true
	}
	if containsField(ctx, "geometry_area") {
		fields.geometryArea = true
	}
	if containsField(ctx, "intersection_geometry") {
		fields.intersectionGeometry = true
	}
	if containsField(ctx, "intersection_area") {
		fields.intersectionArea = true
	}
	return fields
}

func containsField(ctx context.Context, fieldName string) bool {
	fields := graphql.CollectFieldsCtx(ctx, nil)
	for _, field := range fields {
		if field.Name == fieldName {
			return true
		}
	}
	return false
}

func censusDatasetGeographySelect(limit *int, where *model.CensusDatasetGeographyFilter, fields censusGeographySelectFields) sq.SelectBuilder {
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
	if fields.geometry {
		cols = append(cols, "tlcg.geometry as geometry")
	}
	if fields.geometryArea {
		cols = append(cols, "ST_Area(tlcg.geometry) as geometry_area")
	}

	orderBy := sq.Expr("tlcg.id")

	// A normal query..
	q := sq.StatementBuilder.
		Select(cols...).
		From("tl_census_geographies tlcg").
		Join("tl_census_sources tlcs on tlcs.id = tlcg.source_id").
		Join("tl_census_datasets tlcd on tlcd.id = tlcs.dataset_id").
		Join("tl_census_layers tlcl on tlcl.id = tlcg.layer_id").
		Limit(checkLimit(limit))

	if where != nil && where.Location != nil {
		loc := where.Location
		found := true
		var qJoin sq.SelectBuilder
		if loc.Bbox != nil {
			qJoin = sq.StatementBuilder.Select().Column("ST_MakeEnvelope(?,?,?,?,4326) as buffer", loc.Bbox.MinLon, loc.Bbox.MinLat, loc.Bbox.MaxLon, loc.Bbox.MaxLat)
		} else if loc.Within != nil && loc.Within.Valid {
			jj, _ := geojson.Marshal(loc.Within.Val)
			qJoin = sq.StatementBuilder.Select().Column("ST_GeomFromGeoJSON(?) as buffer", string(jj))
		} else if loc.Near != nil {
			radius := checkFloat(&loc.Near.Radius, 0, 1_000_000)
			qJoin = sq.StatementBuilder.Select().Column("ST_Buffer(ST_MakePoint(?,?)::geography, ?) as buffer", loc.Near.Lon, loc.Near.Lat, radius)
		} else if loc.StopBuffer != nil && len(loc.StopBuffer.StopIds) > 0 {
			radius := checkFloat(loc.StopBuffer.Radius, 0, 1_000)
			if radius == 0 {
				qJoin = sq.StatementBuilder.Select().
					Column("gtfs_stops.geometry as buffer").
					From("gtfs_stops").
					Where(In("gtfs_stops.id", loc.StopBuffer.StopIds))

			} else {
				qJoin = sq.StatementBuilder.Select().
					Column("ST_Buffer(ST_Collect(ST_Buffer(gtfs_stops.geometry::geography, ?)::geometry), 0) as buffer", radius).
					From("gtfs_stops").
					Where(In("gtfs_stops.id", loc.StopBuffer.StopIds))
			}
		} else {
			found = false
		}
		if found {
			q = q.JoinClause(qJoin.Prefix("join (").Suffix(") as buffer on true"))
			q = q.Where("ST_Intersects(tlcg.geometry, buffer.buffer)")
			if fields.intersectionArea {
				q = q.Column("ST_Area(ST_Intersection(tlcg.geometry, buffer.buffer)) as intersection_area")
			}
			if fields.intersectionGeometry {
				q = q.Column("ST_Intersection(tlcg.geometry, buffer.buffer) as intersection_geometry")
			}
		}
		if loc.Focus != nil {
			orderBy = sq.Expr("ST_Distance(tlcg.geometry, ST_MakePoint(?,?))", loc.Focus.Lon, loc.Focus.Lat)
		}
	}

	// Check layer, dataset
	if where != nil {
		if where.Layer != nil {
			q = q.Where(sq.Eq{"tlcl.name": where.Layer})
		}
		if where.Search != nil {
			q = q.Where(sq.ILike{"tlcg.name": fmt.Sprintf("%%%s%%", *where.Search)})
		}
		if len(where.Ids) > 0 {
			q = q.Where(sq.Eq{"tlcg.id": where.Ids})
		}
	}

	q = q.OrderByClause(orderBy)
	return q
}

func getBufferStopIds(ctx context.Context, db tldb.Ext, entityType string, entityId int) ([]int, error) {
	// Handle aggregation by entity type
	q := sq.StatementBuilder.
		Select("id").
		Distinct().Options("on (gtfs_stops.id)").
		From("gtfs_stops")
	if entityType == "route" {
		q = q.Join("tl_route_stops ON tl_route_stops.stop_id = gtfs_stops.id").Where(sq.Eq{"tl_route_stops.route_id": entityId})
	} else if entityType == "agency" {
		q = q.Join("tl_route_stops ON tl_route_stops.stop_id = gtfs_stops.id").Where(sq.Eq{"tl_route_stops.agency_id": entityId})
	} else if entityType == "stop" {
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
		Limit(checkLimit(limit)).
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
