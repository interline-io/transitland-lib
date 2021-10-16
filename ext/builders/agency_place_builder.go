package builders

import (
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/tl"
)

type AgencyPlace struct {
	AgencyID string
	tl.MinEntity
	tl.FeedVersionEntity
}

func (rs *AgencyPlace) TableName() string {
	return "tl_agency_places"
}

func (rs *AgencyPlace) Filename() string {
	return "tl_agency_places.txt"
}

////////

type AgencyPlaceBuilder struct {
}

func NewAgencyPlaceBuilder() *AgencyPlaceBuilder {
	return &AgencyPlaceBuilder{}
}

func (pp *AgencyPlaceBuilder) AfterWrite(eid string, ent tl.Entity, emap *tl.EntityMap) error {
	switch v := ent.(type) {
	case *tl.Agency:
		_ = v
	}
	return nil
}

func (pp *AgencyPlaceBuilder) Copy(copier *copier.Copier) error {
	// 	CREATE OR REPLACE FUNCTION tl_generate_agency_places(fvid bigint) RETURNS integer
	//     LANGUAGE plpgsql
	//     AS $_$
	// DECLARE
	//     fvid ALIAS for $1;
	// BEGIN
	// DELETE FROM tl_agency_places WHERE feed_version_id = fvid;
	// INSERT INTO tl_agency_places(feed_version_id, agency_id, count, rank, name, adm1name, adm0name)
	// WITH
	// agency_stops AS (
	//     SELECT agency_id,stop_id FROM tl_route_stops WHERE feed_version_id = fvid GROUP BY (agency_id,stop_id)
	// ),
	// agency_totals AS (
	//     SELECT agency_id,count(*)::numeric FROM agency_stops GROUP BY agency_id
	// ),
	// ne_places AS (
	//     SELECT
	//         gtfs_stops.id AS stop_id,
	//         a.ogc_fid,
	//         count(*) AS count
	//     FROM gtfs_stops
	//     CROSS JOIN LATERAL (
	//         SELECT
	//             ogc_fid AS ogc_fid,
	//             ST_Distance(gtfs_stops.geometry, ne.geometry) AS distance
	//         FROM ne_10m_populated_places ne
	//         ORDER BY gtfs_stops.geometry <-> ne.geometry ASC
	//         LIMIT 1
	//     ) AS a
	//     WHERE feed_version_id = fvid and a.distance < 100000
	//     GROUP BY (gtfs_stops.id,a.ogc_fid)
	// ),
	// ne_admins AS (
	//     select
	//         gtfs_stops.id AS stop_id,
	//         ne.ogc_fid
	//     FROM gtfs_stops
	//     INNER JOIN ne_10m_admin_1_states_provinces ne ON st_intersects(ne.geometry, gtfs_stops.geometry)
	//     WHERE feed_version_id = fvid
	// ),
	// agency_places_group AS (
	//     SELECT
	//         agency_stops.agency_id,
	//         ne_places.ogc_fid,
	//         count(*)
	//     FROM agency_stops
	//     INNER JOIN ne_places ON ne_places.stop_id = agency_stops.stop_id
	//     GROUP BY (agency_stops.agency_id,ne_places.ogc_fid)
	// ),
	// agency_places AS (
	//     SELECT
	//         agency_places_group.agency_id,
	//         agency_places_group.count,
	//         agency_places_group.count / agency_totals.count AS rank,
	//         ne.name,
	//         ne.adm1name,
	//         ne.adm0name
	//     FROM agency_places_group
	//     INNER JOIN ne_10m_populated_places ne ON ne.ogc_fid = agency_places_group.ogc_fid
	//     INNER JOIN agency_totals ON agency_totals.agency_id = agency_places_group.agency_id
	// ),
	// agency_admins_group AS (
	//     SELECT
	//         agency_stops.agency_id,
	//         ne_admins.ogc_fid,
	//         count(*)
	//     FROM agency_stops
	//     INNER JOIN ne_admins ON ne_admins.stop_id = agency_stops.stop_id
	//     GROUP BY (agency_stops.agency_id,ne_admins.ogc_fid)
	// ),
	// agency_admins AS (
	//     select
	//         agency_admins_group.agency_id,
	//         agency_admins_group.count,
	//         agency_admins_group.count / agency_totals.count AS rank,
	//         null,
	//         ne.name,
	//         ne.admin
	//     FROM agency_admins_group
	//     INNER JOIN ne_10m_admin_1_states_provinces ne ON ne.ogc_fid = agency_admins_group.ogc_fid
	//     INNER JOIN agency_totals ON agency_totals.agency_id = agency_admins_group.agency_id
	// ),
	// result AS (
	//     SELECT * FROM agency_places UNION SELECT * FROM agency_admins
	// )
	// SELECT fvid AS feed_version_id, result.* FROM result;
	// RETURN 0;
	// END;
	// $_$;
	bt := []tl.Entity{}
	if _, err := copier.Writer.AddEntities(bt); err != nil {
		return err
	}
	return nil
}
