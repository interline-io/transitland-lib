package find

import (
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/server/model"
)

// This file contains functions that generate squirrel SelectBuilders based on GraphQL filters.

func AgencySelect(limit *int, after *int, ids []int, where *model.AgencyFilter) sq.SelectBuilder {
	q := quickSelect("tl_vw_gtfs_agencies", limit, after, ids)
	if where != nil {
		if where.Search != nil && len(*where.Search) > 1 {
			rank, wc := tsQuery(*where.Search)
			q = q.Column(rank).Where(wc)
		}
		if where.FeedVersionSha1 != nil {
			q = q.Where(sq.Eq{"feed_version_sha1": *where.FeedVersionSha1})
		}
		if where.FeedOnestopID != nil {
			q = q.Where(sq.Eq{"feed_onestop_id": *where.FeedOnestopID})
		}
		if where.AgencyID != nil {
			q = q.Where(sq.Eq{"agency_id": *where.AgencyID})
		}
		if where.AgencyName != nil {
			q = q.Where(sq.Eq{"agency_name": *where.AgencyName})
		}
		if where.OnestopID != nil {
			q = q.Where(sq.Eq{"onestop_id": *where.OnestopID})
		}
		if where.Within != nil && where.Within.Valid {
			q = q.Where("ST_Intersects(t.geometry, ?)", where.Within)
		}
		if where.Near != nil {
			q = q.Where("ST_DWithin(t.geometry, ST_MakePoint(?,?), ?)", where.Near.Lat, where.Near.Lon, where.Near.Radius)
		}
	}
	return q
}

func RouteSelect(limit *int, after *int, ids []int, where *model.RouteFilter) sq.SelectBuilder {
	q := quickSelect("tl_vw_gtfs_routes", limit, after, ids)
	if where != nil {
		if where.Search != nil && len(*where.Search) > 0 {
			rank, wc := tsQuery(*where.Search)
			q = q.Column(rank).Where(wc)
		}
		if len(where.AgencyIds) > 0 {
			q = q.Where(sq.Eq{"agency_id": where.AgencyIds})
		}
		if where.FeedVersionSha1 != nil {
			q = q.Where(sq.Eq{"feed_version_sha1": *where.FeedVersionSha1})
		}
		if where.FeedOnestopID != nil {
			q = q.Where(sq.Eq{"feed_onestop_id": *where.FeedOnestopID})
		}
		if where.RouteID != nil {
			q = q.Where(sq.Eq{"route_id": *where.RouteID})
		}
		if where.OnestopID != nil {
			q = q.Where(sq.Eq{"onestop_id": *where.OnestopID})
		}
		if where.OperatorOnestopID != nil {
			q = q.Where(sq.Eq{"operator_onestop_id": *where.OperatorOnestopID})
		}
		if where.RouteType != nil {
			q = q.Where(sq.Eq{"route_type": where.RouteType})
		}
		if where.Within != nil && where.Within.Valid {
			q = q.Where("ST_Intersects(t.geometry, ?)", where.Within)
		}
		if where.Near != nil {
			q = q.Where("ST_DWithin(t.geometry, ST_MakePoint(?,?), ?)", where.Near.Lat, where.Near.Lon, where.Near.Radius)
		}
	}
	return q
}

func TripSelect(limit *int, after *int, ids []int, where *model.TripFilter) sq.SelectBuilder {
	q := quickSelect("tl_vw_gtfs_trips", limit, after, ids)
	if where != nil {
		if where.FeedVersionSha1 != nil {
			q = q.Where(sq.Eq{"feed_version_sha1": *where.FeedVersionSha1})
		}
		if where.FeedOnestopID != nil {
			q = q.Where(sq.Eq{"feed_onestop_id": *where.FeedOnestopID})
		}
		if where.RouteID != nil {
			q = q.Where(sq.Eq{"route_id": *where.RouteID})
		}
		if where.TripID != nil {
			q = q.Where(sq.Eq{"trip_id": *where.TripID})
		}
		if where.ServiceDate != nil {
			q = q.JoinClause(`
			inner join lateral (
				select gc.id
				from gtfs_calendars gc 
				left join gtfs_calendar_dates gcda on gcda.service_id = gc.id and gcda.exception_type = 1 and gcda.date = ?::date
				left join gtfs_calendar_dates gcdb on gcdb.service_id = gc.id and gcdb.exception_type = 2 and gcdb.date = ?::date
				where 
					gc.id = t.service_id 
					AND ((
						gc.start_date <= ?::date AND gc.end_date >= ?::date
						AND (CASE EXTRACT(isodow FROM ?::date)
						WHEN 1 THEN monday = 1
						WHEN 2 THEN tuesday = 1
						WHEN 3 THEN wednesday = 1
						WHEN 4 THEN thursday = 1
						WHEN 5 THEN friday = 1
						WHEN 6 THEN saturday = 1
						WHEN 7 THEN sunday = 1
						END)
					) OR gcda.date IS NOT NULL)
					AND gcdb.date is null
				LIMIT 1
			) gc on true
			`, where.ServiceDate, where.ServiceDate, where.ServiceDate, where.ServiceDate, where.ServiceDate)
		}
	}
	return q
}

func StopSelect(limit *int, after *int, ids []int, where *model.StopFilter) sq.SelectBuilder {
	q := quickSelect("tl_vw_gtfs_stops", limit, after, ids)
	if where != nil {
		if where.Search != nil && len(*where.Search) > 1 {
			rank, wc := tsQuery(*where.Search)
			q = q.Column(rank).Where(wc)
		}
		if where.FeedOnestopID != nil {
			q = q.Where(sq.Eq{"feed_onestop_id": *where.FeedOnestopID})
		}
		if where.FeedVersionSha1 != nil {
			q = q.Where(sq.Eq{"feed_version_sha1": *where.FeedVersionSha1})
		}
		if where.OnestopID != nil {
			q = q.Where(sq.Eq{"onestop_id": *where.OnestopID})
		}
		if where.StopID != nil {
			q = q.Where(sq.Eq{"stop_id": *where.StopID})
		}
		if len(where.AgencyIds) > 0 {
			q = q.Join("tl_route_stops on tl_route_stops.stop_id = t.id").Where(sq.Eq{"tl_route_stops.agency_id": where.AgencyIds}).Distinct().Options("on (t.id)")
		}
		if where.Within != nil && where.Within.Valid {
			q = q.Where("ST_Intersects(t.geometry, ?)", where.Within)
		}
		if where.Near != nil {
			q = q.Where("ST_DWithin(t.geometry, ST_MakePoint(?,?), ?)", where.Near.Lat, where.Near.Lon, where.Near.Radius)
		}
	}
	return q
}

func OperatorSelect(limit *int, after *int, ids []int, where *model.OperatorFilter) sq.SelectBuilder {
	q := quickSelectOrder("tl_mv_active_agency_operators", limit, after, ids, "")
	if where != nil {
		if where.Search != nil && len(*where.Search) > 0 {
			rank, wc := tsQuery(*where.Search)
			q = q.Column(rank).Where(wc)
		}
		if where.Merged != nil && *where.Merged == true {
			q = q.Distinct().Options("on (onestop_id)")
		} else {
			q = q.OrderBy("id asc")
		}
		if where.FeedVersionSha1 != nil {
			q = q.Where(sq.Eq{"feed_version_sha1": *where.FeedVersionSha1})
		}
		if where.FeedOnestopID != nil {
			q = q.Where(sq.Eq{"feed_onestop_id": *where.FeedOnestopID})
		}
		if where.AgencyID != nil {
			q = q.Where(sq.Eq{"agency_id": *where.AgencyID})
		}
		if where.OnestopID != nil {
			q = q.Where(sq.Eq{"onestop_id": where.OnestopID})
		}
	}
	return q
}

func PathwaySelect(limit *int, after *int, ids []int, where *model.PathwayFilter) sq.SelectBuilder {
	q := quickSelectOrder("gtfs_pathways", limit, after, ids, "")
	if where != nil {
		if where.PathwayMode != nil {
			q = q.Where(sq.Eq{"pathway_mode": where.PathwayMode})
		}
	}
	return q
}
func FeedVersionSelect(limit *int, after *int, ids []int, where *model.FeedVersionFilter) sq.SelectBuilder {
	q := quickSelectOrder("feed_versions", limit, after, ids, "")
	q = q.Join("current_feeds cf on cf.id = t.feed_id").Where(sq.Eq{"cf.deleted_at": nil})
	q = q.OrderBy("fetched_at desc")
	if where != nil {
		if where.Sha1 != nil {
			q = q.Where(sq.Eq{"sha1": *where.Sha1})
		}
		if where.FeedOnestopID != nil {
			q = q.Where(sq.Eq{"cf.onestop_id": *where.FeedOnestopID})
		}
	}
	return q
}

func FeedSelect(limit *int, after *int, ids []int, where *model.FeedFilter) sq.SelectBuilder {
	q := quickSelect("current_feeds", limit, after, ids)
	q = q.Where(sq.Eq{"deleted_at": nil})
	if where != nil {
		if where.Search != nil && len(*where.Search) > 1 {
			rank, wc := tsQuery(*where.Search)
			q = q.Column(rank).Where(wc)
		}
		if where.OnestopID != nil {
			q = q.Where(sq.Eq{"onestop_id": *where.OnestopID})
		}
		if len(where.Spec) > 0 {
			q = q.Where(sq.Eq{"spec": where.Spec})
		}
		// Fetch error
		if v := where.FetchError; v == nil {
			// nothing
		} else if *v == true {
			q = q.Join("feed_states on feed_states.feed_id = t.id").Where(sq.NotEq{"feed_states.last_fetch_error": ""})
		} else if *v == false {
			q = q.Join("feed_states on feed_states.feed_id = t.id").Where(sq.Eq{"feed_states.last_fetch_error": ""})
		}
		// Import import status
		if where.ImportStatus != nil {
			// in_progress must be false to check success and vice-versa
			var checkSuccess bool
			var checkInProgress bool
			check := *where.ImportStatus
			if check == "success" {
				checkSuccess = true
				checkInProgress = false
			} else if check == "error" {
				checkSuccess = false
				checkInProgress = false
			} else if check == "in_progress" {
				checkSuccess = false
				checkInProgress = true
			}
			// This lateral join gets the most recent attempt at a completed feed_version_gtfs_import and checks the status
			q = q.JoinClause("JOIN LATERAL (select fvi.in_progress, fvi.success from feed_versions fv inner join feed_version_gtfs_imports fvi on fvi.feed_version_id = fv.id WHERE fv.feed_id = t.id ORDER BY fvi.id DESC LIMIT 1) fvicheck ON TRUE").
				Where(sq.Eq{"fvicheck.success": checkSuccess, "fvicheck.in_progress": checkInProgress})
		}
	}
	return q
}

func FeedVersionServiceLevelSelect(limit *int, after *int, ids []int, where *model.FeedVersionServiceLevelFilter) sq.SelectBuilder {
	q := quickSelectOrder("feed_version_service_levels", limit, after, nil, "")
	if where == nil {
		where = &model.FeedVersionServiceLevelFilter{}
	}
	if where.DistinctOn != nil {
		q = q.Distinct().Options("ON (feed_version_id,route_id)").OrderBy("feed_version_id,route_id")
	} else {
		q = q.OrderBy("id")
	}
	if where.AllRoutes != nil && *where.AllRoutes == true {
		// default
	} else if len(where.RouteIds) > 0 {
		q = q.Where(sq.Eq{"route_id": where.RouteIds})
	} else {
		q = q.Where(sq.Eq{"route_id": nil})
	}
	if where.StartDate != nil {
		q = q.Where(sq.GtOrEq{"start_date": where.StartDate})
	}
	if where.EndDate != nil {
		q = q.Where(sq.LtOrEq{"end_date": where.EndDate})
	}
	return q
}

func AgencyPlaceSelect(limit *int, after *int, ids []int, where *model.AgencyPlaceFilter) sq.SelectBuilder {
	q := quickSelect("tl_agency_places", limit, after, ids)
	if where != nil {
		// if where.Search != nil && len(*where.Search) > 1 {
		// 	q = q.Where(tsQuery(*where.Search))
		// }
		if where.MinRank != nil {
			q = q.Where(sq.GtOrEq{"rank": where.MinRank})
		}
	}
	return q
}

func CensusGeographySelect(param *model.CensusGeographyParam, eids []int) sq.SelectBuilder {
	if param.EntityID > 0 {
		eids = append(eids, param.EntityID)
	}
	r := checkFloat(param.Radius, 0, 2000.0)
	// Include matched entity column
	s := "gtfs_stops.id as match_entity_id, t.*"
	if param.EntityType == "route" {
		s = "tl_route_stops.route_id as match_entity_id, t.*"
	} else if param.EntityType == "agency" {
		s = "tl_route_stops.agency_id as match_entity_id, t.*"
	}
	// A normal query..
	q := sq.StatementBuilder.Select(s).From("tl_census_geographies t").
		InnerJoin("gtfs_stops ON ST_DWithin(t.geometry, gtfs_stops.geometry, ?)", r).
		Where(sq.Eq{"t.layer_name": param.LayerName}).
		Limit(checkLimit(param.Limit))
	// Handle aggregation by entity type
	if param.EntityType == "route" {
		q = q.InnerJoin("tl_route_stops ON tl_route_stops.stop_id = gtfs_stops.id")
		q = q.Distinct().Options("on (tl_route_stops.route_id,t.id)").Where(sq.Eq{"tl_route_stops.route_id": eids}).OrderBy("tl_route_stops.route_id,t.id")
	} else if param.EntityType == "agency" {
		q = q.InnerJoin("tl_route_stops ON tl_route_stops.stop_id = gtfs_stops.id")
		q = q.Distinct().Options("on (tl_route_stops.stop_id,t.id)").Where(sq.Eq{"tl_route_stops.agency_id": eids}).OrderBy("tl_route_stops.stop_id,t.id")
	} else if param.EntityType == "stop" {
		q = q.Where(sq.Eq{"gtfs_stops.id": eids}).OrderBy("id")
	}
	return q
}

func CensusValueSelect(param *model.CensusValueParam, eids []int) sq.SelectBuilder {
	if param.GeographyID > 0 {
		eids = append(eids, param.GeographyID)
	}
	tnames := strings.Split(param.TableNames, ",")
	q := quickSelectOrder("tl_census_values", param.Limit, nil, nil, "").
		InnerJoin("tl_census_tables ON tl_census_tables.id = t.table_id").
		Where(sq.Eq{"t.geography_id": eids}).
		Where(sq.Eq{"tl_census_tables.table_name": tnames})
	return q
}

func RouteStopBufferSelect(param model.RouteStopBufferParam) sq.SelectBuilder {
	r := checkFloat(param.Radius, 0, 2000.0)
	q := sq.StatementBuilder.
		Select(
			"ST_Collect(gtfs_stops.geometry::geometry)::geography AS stop_points",
			"ST_ConvexHull(ST_Collect(gtfs_stops.geometry::geometry))::geography AS stop_convexhull",
		).
		Column(sq.Expr("ST_Buffer(ST_Collect(gtfs_stops.geometry::geometry)::geography, ?, 4)::geography AS stop_buffer", r)). // column expr
		From("gtfs_stops").
		InnerJoin("tl_route_stops on tl_route_stops.stop_id = gtfs_stops.id").
		Where(sq.Eq{"tl_route_stops.route_id": param.EntityID})
	return q
}
