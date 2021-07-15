package find

import (
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/server/model"
)

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
