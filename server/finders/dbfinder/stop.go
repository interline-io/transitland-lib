package dbfinder

import (
	"context"
	"encoding/json"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
	sq "github.com/irees/squirrel"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

func (f *Finder) FindStops(ctx context.Context, limit *int, after *model.Cursor, ids []int, where *model.StopFilter) ([]*model.Stop, error) {
	var ents []*model.Stop
	active := true
	if len(ids) > 0 || (where != nil && where.FeedVersionSha1 != nil) {
		active = false
	}
	q := stopSelect(limit, after, ids, active, f.PermFilter(ctx), where)
	if err := dbutil.Select(ctx, f.db, q, &ents); err != nil {
		return nil, logErr(ctx, err)
	}
	return ents, nil
}

func (f *Finder) StopExternalReferencesByStopIDs(ctx context.Context, ids []int) ([]*model.StopExternalReference, []error) {
	var ents []*model.StopExternalReference
	q := sq.StatementBuilder.Select("*").From("tl_stop_external_references").Where(In("stop_id", ids))
	if err := dbutil.Select(ctx, f.db, q, &ents); err != nil {
		return nil, []error{err}
	}
	return arrangeBy(ids, ents, func(ent *model.StopExternalReference) int { return ent.StopID.Int() }), nil
}

func (f *Finder) StopObservationsByStopIDs(ctx context.Context, limit *int, where *model.StopObservationFilter, keys []int) ([][]*model.StopObservation, error) {
	// Prepare output
	q := sq.StatementBuilder.Select("gtfs_stops.id as stop_id", "obs.*").
		From("ext_performance_stop_observations obs").
		Join("gtfs_stops on gtfs_stops.stop_id = obs.to_stop_id").
		Where(In("gtfs_stops.id", keys)).
		Limit(finderCheckLimitMax(limit, FINDER_STOP_OBSERVATION_MAXLIMIT))
	if where != nil {
		q = q.Where("obs.feed_version_id = ?", where.FeedVersionID)
		q = q.Where("obs.trip_start_date = ?", where.TripStartDate)
		q = q.Where("obs.source = ?", where.Source)
		// q = q.Where("start_time >= ?", where.StartTime)
		// q = q.Where("end_time <= ?", where.EndTime)
	}
	var ents []*model.StopObservation
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.StopObservation) int { return ent.StopID }), err
}

func (f *Finder) StopsByIDs(ctx context.Context, ids []int) ([]*model.Stop, []error) {
	ents, err := f.FindStops(ctx, nil, nil, ids, nil)
	if err != nil {
		return nil, logExtendErr(ctx, len(ids), err)
	}
	return arrangeBy(ids, ents, func(ent *model.Stop) int { return ent.ID }), nil
}

func (f *Finder) StopsByRouteIDs(ctx context.Context, limit *int, where *model.StopFilter, keys []int) ([][]*model.Stop, error) {
	var ents []*model.Stop
	qso := stopSelect(limit, nil, nil, false, f.PermFilter(ctx), where)
	qso = qso.Join("tl_route_stops on tl_route_stops.stop_id = gtfs_stops.id").Where(In("route_id", keys)).Column("route_id as with_route_id")
	err := dbutil.Select(ctx,
		f.db,
		qso,
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.Stop) int { return ent.WithRouteID.Int() }), err
}

func (f *Finder) StopsByParentStopIDs(ctx context.Context, limit *int, where *model.StopFilter, keys []int) ([][]*model.Stop, error) {
	var ents []*model.Stop
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			stopSelect(limit, nil, nil, false, f.PermFilter(ctx), where),
			"gtfs_stops",
			"id",
			"gtfs_stops",
			"parent_station",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.Stop) int { return ent.ParentStation.Int() }), err
}

func (f *Finder) TargetStopsByStopIDs(ctx context.Context, ids []int) ([]*model.Stop, []error) {
	if len(ids) == 0 {
		return nil, nil
	}
	// TODO: this is moderately cursed
	type qlookup struct {
		SourceID int
		*model.Stop
	}
	var qents []*qlookup
	q := sq.
		Select("t.*", "tlse.stop_id as source_id").
		FromSelect(stopSelect(nil, nil, nil, true, f.PermFilter(ctx), nil), "t").
		Join("tl_stop_external_references tlse on tlse.target_feed_onestop_id = t.feed_onestop_id and tlse.target_stop_id = t.stop_id").
		Where(In("tlse.stop_id", ids))
	if err := dbutil.Select(ctx,
		f.db,
		q,
		&qents,
	); err != nil {
		return nil, logExtendErr(ctx, 0, err)
	}
	group := map[int]*model.Stop{}
	for _, ent := range qents {
		group[ent.SourceID] = ent.Stop
	}
	var ents []*model.Stop
	for _, id := range ids {
		ents = append(ents, group[id])
	}
	return ents, nil
}

func (f *Finder) StopsByFeedVersionIDs(ctx context.Context, limit *int, where *model.StopFilter, keys []int) ([][]*model.Stop, error) {
	var ents []*model.Stop
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			stopSelect(limit, nil, nil, false, f.PermFilter(ctx), where),
			"feed_versions",
			"id",
			"gtfs_stops",
			"feed_version_id",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.Stop) int { return ent.FeedVersionID }), err
}

func (f *Finder) StopsByLevelIDs(ctx context.Context, limit *int, where *model.StopFilter, keys []int) ([][]*model.Stop, error) {
	var ents []*model.Stop
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			stopSelect(limit, nil, nil, false, f.PermFilter(ctx), where),
			"gtfs_levels",
			"id",
			"gtfs_stops",
			"level_id",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.Stop) int { return ent.LevelID.Int() }), err
}

func (f *Finder) StopPlacesByStopID(ctx context.Context, params []model.StopPlaceParam) ([]*model.StopPlace, []error) {
	if f.adminCache == nil {
		return f.stopPlacesByStopIdFallback(ctx, params)
	}

	// Lookup any geometries that were not passed in
	var geomLookup []int
	for _, param := range params {
		if param.Point.Lon == 0 && param.Point.Lat == 0 {
			geomLookup = append(geomLookup, param.ID)
		}
	}
	if len(geomLookup) > 0 {
		var ents []struct {
			ID       int
			Geometry tt.Point
		}
		q := sq.Select("gtfs_stops.id", "gtfs_stops.geometry").From("gtfs_stops").Where(sq.Eq{"id": geomLookup})
		if err := dbutil.Select(ctx, f.db, q, &ents); err != nil {
			return nil, logExtendErr(ctx, len(params), err)
		}
		lk := map[int]tlxy.Point{}
		for _, ent := range ents {
			lk[ent.ID] = tlxy.Point{Lon: ent.Geometry.X(), Lat: ent.Geometry.Y()}
		}
		for i := 0; i < len(params); i++ {
			if pt, ok := lk[params[i].ID]; ok {
				params[i].Point = pt
			}
		}
	}

	// Lookup stop places using adminCache
	a := map[int]*model.StopPlace{}
	for _, param := range params {
		if admin, ok := f.adminCache.Check(param.Point); ok {
			a[param.ID] = &model.StopPlace{
				Adm0Name: &admin.Adm0Name,
				Adm1Name: &admin.Adm1Name,
				Adm0Iso:  &admin.Adm0Iso,
				Adm1Iso:  &admin.Adm1Iso,
			}
		}
	}
	ret := make([]*model.StopPlace, len(params))
	for idx, param := range params {
		ret[idx] = a[param.ID]
	}
	return ret, nil
}

func (f *Finder) stopPlacesByStopIdFallback(ctx context.Context, params []model.StopPlaceParam) ([]*model.StopPlace, []error) {
	// Fallback without adminCache
	var ids []int
	for _, param := range params {
		ids = append(ids, param.ID)
	}
	type result struct {
		StopID int
		model.StopPlace
	}
	var ents []result
	detailedQuery := sq.Select("gtfs_stops.id as stop_id", "ne.name as adm1_name", "ne.admin as adm0_name").
		From("ne_10m_admin_1_states_provinces ne").
		Join("gtfs_stops on ST_Intersects(gtfs_stops.geometry, ne.geometry)").
		Where(In("gtfs_stops.id", ids))
	if err := dbutil.Select(ctx, f.db, detailedQuery, &ents); err != nil {
		return nil, logExtendErr(ctx, len(ids), err)
	}
	return arrangeMap(ids, ents, func(ent result) (int, *model.StopPlace) { return ent.StopID, &ent.StopPlace }), nil
}

func stopSelect(limit *int, after *model.Cursor, ids []int, active bool, permFilter *model.PermFilter, where *model.StopFilter) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_stops.id",
		"gtfs_stops.feed_version_id",
		"gtfs_stops.stop_id",
		"gtfs_stops.stop_code",
		"gtfs_stops.stop_desc",
		"gtfs_stops.stop_name",
		"gtfs_stops.stop_timezone",
		"gtfs_stops.stop_url",
		"gtfs_stops.location_type",
		"gtfs_stops.wheelchair_boarding",
		"gtfs_stops.zone_id",
		"gtfs_stops.platform_code",
		"gtfs_stops.tts_stop_name",
		"gtfs_stops.geometry",
		"gtfs_stops.level_id",
		"gtfs_stops.parent_station",
		"gtfs_stops.area_id",
		"current_feeds.id AS feed_id",
		"current_feeds.onestop_id AS feed_onestop_id",
		"feed_versions.sha1 AS feed_version_sha1",
		"coalesce(feed_version_stop_onestop_ids.onestop_id, '') as onestop_id",
	).
		From("gtfs_stops").
		Join("feed_versions ON feed_versions.id = gtfs_stops.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id").
		OrderBy("gtfs_stops.feed_version_id,gtfs_stops.id").
		Limit(finderCheckLimit(limit))
	distinct := false

	// Handle previous OnestopIds
	if where != nil {
		// Allow either a single onestop id or multiple
		if where.OnestopID != nil {
			where.OnestopIds = append(where.OnestopIds, *where.OnestopID)
		}
		if len(where.OnestopIds) > 0 && where.AllowPreviousOnestopIds != nil && *where.AllowPreviousOnestopIds {
			// Use CTE for stop lookup optimization
			sub := sq.StatementBuilder.
				Select("feed_version_stop_onestop_ids.onestop_id", "feed_version_stop_onestop_ids.entity_id", "feed_versions.feed_id").
				Distinct().Options("on (feed_version_stop_onestop_ids.onestop_id, feed_version_stop_onestop_ids.entity_id, feed_versions.feed_id)").
				From("feed_version_stop_onestop_ids").
				Join("feed_versions on feed_versions.id = feed_version_stop_onestop_ids.feed_version_id").
				Where(In("feed_version_stop_onestop_ids.onestop_id", where.OnestopIds)).
				OrderBy("feed_version_stop_onestop_ids.onestop_id, feed_version_stop_onestop_ids.entity_id, feed_versions.feed_id, feed_versions.id DESC")
			stopLookupCte := sq.CTE{
				Materialized: true,
				Alias:        "feed_version_stop_onestop_ids",
				Expression:   sub,
			}
			q = q.
				WithCTE(stopLookupCte).
				Join("feed_version_stop_onestop_ids on feed_version_stop_onestop_ids.entity_id = gtfs_stops.stop_id and feed_version_stop_onestop_ids.feed_id = feed_versions.feed_id")
		} else {
			q = q.JoinClause(`LEFT JOIN feed_version_stop_onestop_ids ON feed_version_stop_onestop_ids.entity_id = gtfs_stops.stop_id and feed_version_stop_onestop_ids.feed_version_id = gtfs_stops.feed_version_id`)
			if len(where.OnestopIds) > 0 {
				q = q.Where(In("feed_version_stop_onestop_ids.onestop_id", where.OnestopIds))
			}
		}
	} else {
		q = q.JoinClause(`LEFT JOIN feed_version_stop_onestop_ids ON feed_version_stop_onestop_ids.entity_id = gtfs_stops.stop_id and feed_version_stop_onestop_ids.feed_version_id = gtfs_stops.feed_version_id`)
	}

	// Handle geom search
	if where != nil {
		loc := where.Location
		if loc == nil {
			loc = &model.StopLocationFilter{
				Bbox:    where.Bbox,
				Near:    where.Near,
				Polygon: where.Within,
			}
		}
		if len(loc.Features) > 0 {
			// Set bounding box from features
			var fc []*geojson.Feature
			fcBbox := geom.Bounds{}
			for _, f := range loc.Features {
				fc = append(fc, &geojson.Feature{
					ID:       nilOr(f.ID, ""),
					Geometry: f.Geometry.Val,
				})
				fcBbox.Extend(f.Geometry.Val)
			}
			fcBbox2 := &model.BoundingBox{
				MinLon: fcBbox.Min(0),
				MinLat: fcBbox.Min(1),
				MaxLon: fcBbox.Max(0),
				MaxLat: fcBbox.Max(1),
			}

			// Search based on GeoJSON features, and include the matching features in response
			fjArray, err := json.Marshal(fc)
			if err != nil {
				log.Error().Msgf("failed to encode features as json: %s", err)
			}
			featureData := sq.StatementBuilder.
				Select(
					"feature->>'id' as feature_id",
					"ST_GeomFromGeoJSON(feature->'geometry') geometry",
				).
				FromSelect(sq.StatementBuilder.Select().Column(sq.Expr("json_array_elements(?::json) feature", string(fjArray))), "t")

			// Must be json_agg to work with tt.Strings
			featureQuery := sq.StatementBuilder.
				Select(
					"gtfs_stops.id",
					"json_agg(features.feature_id) feature_ids",
				).
				From("gtfs_stops").
				JoinClause(featureData.Prefix("JOIN (").Suffix(") features ON ST_Intersects(gtfs_stops.geometry, features.geometry)")).
				GroupBy("gtfs_stops.id")
			q = q.
				Column("features.feature_ids as within_features").
				JoinClause(featureQuery.Prefix("JOIN (").Suffix(") features on features.id = gtfs_stops.id")).
				Where("ST_Intersects(gtfs_stops.geometry, ST_MakeEnvelope(?,?,?,?,4326))", fcBbox2.MinLon, fcBbox2.MinLat, fcBbox2.MaxLon, fcBbox2.MaxLat)
		}
		if len(loc.GeographyIds) > 0 {
			featureData := sq.StatementBuilder.
				Select(
					"tlcg.id::text as feature_id",
					"tlcg.geometry",
				).
				From("tl_census_geographies tlcg").
				Where(sq.Eq{"tlcg.id": loc.GeographyIds})
			featureQuery := sq.StatementBuilder.
				Select(
					"gtfs_stops.id",
					"json_agg(features.feature_id) feature_ids",
				).
				From("gtfs_stops").
				JoinClause(featureData.Prefix("JOIN (").Suffix(") features ON ST_Intersects(gtfs_stops.geometry, features.geometry)")).
				GroupBy("gtfs_stops.id")
			q = q.
				Column("features.feature_ids as within_features").
				JoinClause(featureQuery.Prefix("JOIN (").Suffix(") features on features.id = gtfs_stops.id"))
		}
		if loc.Bbox != nil {
			q = q.Where("ST_Intersects(gtfs_stops.geometry, ST_MakeEnvelope(?,?,?,?,4326))", loc.Bbox.MinLon, loc.Bbox.MinLat, loc.Bbox.MaxLon, loc.Bbox.MaxLat)
		}
		if loc.Polygon != nil && loc.Polygon.Valid {
			q = q.Where("ST_Intersects(gtfs_stops.geometry, ?)", loc.Polygon)
		}
		if loc.Near != nil {
			radius := checkFloat(&loc.Near.Radius, 0, 1_000_000)
			q = q.Where("ST_DWithin(gtfs_stops.geometry, ST_MakePoint(?,?), ?)", loc.Near.Lon, loc.Near.Lat, radius)
		}
	}

	// Handle other clauses
	if where != nil {
		if where.StopCode != nil {
			q = q.Where(sq.Eq{"gtfs_stops.stop_code": where.StopCode})
		}
		if where.LocationType != nil {
			q = q.Where(sq.Eq{"gtfs_stops.location_type": where.LocationType})
		}
		if where.FeedOnestopID != nil {
			q = q.Where(sq.Eq{"current_feeds.onestop_id": *where.FeedOnestopID})
		}
		if where.FeedVersionSha1 != nil {
			q = q.Where("feed_versions.id = (select id from feed_versions where sha1 = ? limit 1)", *where.FeedVersionSha1)
		}
		if where.StopID != nil {
			q = q.Where(sq.Eq{"gtfs_stops.stop_id": *where.StopID})
		}
		if where.Serviced != nil {
			q = q.JoinClause(`left join lateral (select tlrs_serviced.stop_id from tl_route_stops tlrs_serviced where tlrs_serviced.stop_id = gtfs_stops.id limit 1) scount on true`)
			if *where.Serviced {
				q = q.Where(sq.NotEq{"scount.stop_id": nil})
			} else {
				q = q.Where(sq.Eq{"scount.stop_id": nil})
			}
		}

		// Served by agency ID
		if len(where.AgencyIds) > 0 {
			distinct = true
			q = q.Join("tl_route_stops tlrs_agencies on tlrs_agencies.stop_id = gtfs_stops.id").Where(In("tlrs_agencies.agency_id", where.AgencyIds))
		}

		// Served by route type
		if where.ServedByRouteType != nil {
			where.ServedByRouteTypes = append(where.ServedByRouteTypes, *where.ServedByRouteType)
		}
		if len(where.ServedByRouteTypes) > 0 {
			q = q.JoinClause(
				`join lateral (select tlrs_rt.stop_id from tl_route_stops tlrs_rt join gtfs_routes on gtfs_routes.id = tlrs_rt.route_id where tlrs_rt.stop_id = gtfs_stops.id and gtfs_routes.route_type = ANY(?) limit 1) rt on true`,
				where.ServedByRouteTypes,
			)
		}

		// Accepts both route and operator Onestop IDs
		if len(where.ServedByOnestopIds) > 0 {
			distinct = true
			agencies := []string{}
			routes := []string{}
			for _, osid := range where.ServedByOnestopIds {
				if len(osid) == 0 {
				} else if osid[0] == 'o' {
					agencies = append(agencies, osid)
				} else if osid[0] == 'r' {
					routes = append(routes, osid)
				}
			}
			q = q.
				Join("tl_route_stops tlrs_routes on tlrs_routes.stop_id = gtfs_stops.id").
				Join("gtfs_routes on gtfs_routes.id = tlrs_routes.route_id")
			if len(routes) > 0 {
				q = q.Join("feed_version_route_onestop_ids on gtfs_routes.route_id = feed_version_route_onestop_ids.entity_id and gtfs_stops.feed_version_id = feed_version_route_onestop_ids.feed_version_id")
			}
			if len(agencies) > 0 {
				q = q.
					Join("gtfs_agencies on gtfs_agencies.id = tlrs_routes.agency_id").
					Join("current_operators_in_feed coif ON coif.resolved_gtfs_agency_id = gtfs_agencies.agency_id AND coif.feed_id = current_feeds.id")
			}
			if len(routes) > 0 && len(agencies) > 0 {
				q = q.Where(sq.Or{
					In("feed_version_route_onestop_ids.onestop_id", routes),
					In("coif.resolved_onestop_id", agencies),
				})
			} else if len(routes) > 0 {
				q = q.Where(In("feed_version_route_onestop_ids.onestop_id", routes))
			} else if len(agencies) > 0 {
				q = q.Where(In("coif.resolved_onestop_id", agencies))
			}
		}

		// Handle license filtering
		q = licenseFilter(where.License, q)

		// Text search
		if where.Search != nil && len(*where.Search) > 1 {
			rank, wc := tsTableQuery("gtfs_stops", *where.Search)
			q = q.Column(rank).Where(wc)
		}
	}

	if distinct {
		q = q.Distinct().Options("on (gtfs_stops.feed_version_id,gtfs_stops.id)")
	}
	if active {
		q = q.Join("feed_states on feed_states.feed_version_id = gtfs_stops.feed_version_id")
	}
	if len(ids) > 0 {
		q = q.Where(In("gtfs_stops.id", ids))
	}

	// Handle cursor
	if after != nil && after.Valid && after.ID > 0 {
		// first check helps improve query performance
		if after.FeedVersionID == 0 {
			q = q.
				Where(sq.Expr("gtfs_stops.feed_version_id >= (select feed_version_id from gtfs_stops where id = ?)", after.ID)).
				Where(sq.Expr("(gtfs_stops.feed_version_id, gtfs_stops.id) > (select feed_version_id,id from gtfs_stops where id = ?)", after.ID))
		} else {
			q = q.
				Where(sq.Expr("gtfs_stops.feed_version_id >= ?", after.FeedVersionID)).
				Where(sq.Expr("(gtfs_stops.feed_version_id, gtfs_stops.id) > (?,?)", after.FeedVersionID, after.ID))
		}
	}

	// Handle permissions
	q = pfJoinCheckFv(q, permFilter)
	return q
}
