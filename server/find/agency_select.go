package find

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/jmoiron/sqlx"
)

func FindAgencies(atx sqlx.Ext, limit *int, after *int, ids []int, where *model.AgencyFilter) (ents []*model.Agency, err error) {
	q := AgencySelect(limit, after, ids, where)
	if len(ids) == 0 && (where == nil || where.FeedVersionSha1 == nil) {
		q = q.Where(sq.NotEq{"active": nil})
	}
	MustSelect(model.DB, q, &ents)
	return ents, nil
}

func AgencySelect(limit *int, after *int, ids []int, where *model.AgencyFilter) sq.SelectBuilder {
	qView := sq.StatementBuilder.
		Select(
			"gtfs_agencies.*",
			"tl_agency_geometries.geometry",
			"tl_agency_geometries.centroid",
			"tlp.name AS city_name",
			"tlp.adm1name AS adm1name",
			"tlp.adm0name AS adm0name",
			"current_feeds.id AS feed_id",
			"current_feeds.onestop_id AS feed_onestop_id",
			"feed_versions.sha1 AS feed_version_sha1",
			`COALESCE(
				coif.onestop_id, 
				tl_agency_onestop_ids.onestop_id::character varying, 
				(
					(('o-'::text || "right"(current_feeds.onestop_id::text, length(current_feeds.onestop_id::text) - 2)) || 
					'-'::text) || 
					regexp_replace(regexp_replace(lower(gtfs_agencies.agency_name), '[\-\:\&\@\/]', '~', 'g'), '[^[:alnum:]\~\>\<]', '', 'g')
				)::character varying
			) AS onestop_id`,
			"feed_states.feed_version_id AS active",
		).
		From("gtfs_agencies").
		Join("feed_versions ON feed_versions.id = gtfs_agencies.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id").
		JoinClause("LEFT JOIN tl_agency_geometries ON tl_agency_geometries.agency_id = gtfs_agencies.id").
		JoinClause("LEFT JOIN tl_agency_onestop_ids ON tl_agency_onestop_ids.agency_id = gtfs_agencies.id").
		JoinClause("LEFT JOIN feed_states ON feed_states.feed_version_id = gtfs_agencies.feed_version_id").
		JoinClause(`LEFT JOIN LATERAL (
			SELECT co.onestop_id
			FROM current_operators co
			JOIN current_operators_in_feed coif_1 ON coif_1.operator_id = co.id
			WHERE co.deleted_at IS NULL AND coif_1.feed_id = current_feeds.id AND coif_1.gtfs_agency_id::text = gtfs_agencies.agency_id::text
			ORDER BY co.onestop_id
			LIMIT 1) coif ON true`).
		JoinClause(`LEFT JOIN LATERAL ( 
			SELECT tlp_1.name, tlp_1.adm1name, tlp_1.adm0name
			FROM tl_agency_places tlp_1
			WHERE tlp_1.agency_id = gtfs_agencies.id AND tlp_1.rank > 0.2::double precision
			ORDER BY tlp_1.rank DESC
			LIMIT 1) tlp ON true`).
		Where(sq.Eq{"current_feeds.deleted_at": nil})

	q := sq.StatementBuilder.Select("*").FromSelect(qView, "t")
	if len(ids) > 0 {
		q = q.Where(sq.Eq{"t.id": ids})
	}
	if after != nil {
		q = q.Where(sq.Gt{"t.id": *after})
	}
	q = q.Limit(checkLimit(limit))
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
