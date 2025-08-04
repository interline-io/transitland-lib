package dbfinder

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/model"
	"github.com/interline-io/transitland-mw/dbutil"
)

func (f *Finder) FindOperators(ctx context.Context, limit *int, after *model.Cursor, ids []int, where *model.OperatorFilter) ([]*model.Operator, error) {
	var ents []*model.Operator
	if err := dbutil.Select(ctx, f.db, operatorSelect(limit, after, ids, f.PermFilter(ctx), where), &ents); err != nil {
		return nil, logErr(ctx, err)
	}
	return ents, nil
}

func (f *Finder) OperatorsByCOIFs(ctx context.Context, ids []int) ([]*model.Operator, []error) {
	var ents []*model.Operator
	err := dbutil.Select(ctx,
		f.db,
		operatorSelect(nil, nil, ids, f.PermFilter(ctx), nil),
		&ents,
	)
	if err != nil {
		return nil, logExtendErr(ctx, len(ids), err)
	}
	return arrangeBy(ids, ents, func(ent *model.Operator) int { return ent.ID }), nil
}

func (f *Finder) OperatorsByAgencyIDs(ctx context.Context, ids []int) ([]*model.Operator, []error) {
	var ents []*model.Operator
	err := dbutil.Select(ctx,
		f.db,
		operatorsByAgencyID(nil, nil, ids),
		&ents,
	)
	if err != nil {
		return nil, logExtendErr(ctx, len(ids), err)
	}
	return arrangeBy(ids, ents, func(ent *model.Operator) int { return ent.AgencyID }), nil
}

// Param loaders

func (f *Finder) OperatorsByFeedIDs(ctx context.Context, limit *int, where *model.OperatorFilter, keys []int) ([][]*model.Operator, error) {
	var ents []*model.Operator
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			operatorSelectBase(true, nil),
			"current_feeds",
			"id",
			"coif",
			"feed_id",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.Operator) int { return ent.FeedID }), err
}

func operatorSelectBase(distinct bool, where *model.OperatorFilter) sq.SelectBuilder {
	q := sq.StatementBuilder.
		Select(
			"coif.id as id",
			"coif.feed_id as feed_id",
			"coif.resolved_name as name",
			"coif.resolved_short_name as short_name",
			"coif.resolved_onestop_id as onestop_id",
			"coif.textsearch as textsearch",
			"current_feeds.onestop_id as feed_onestop_id",
			"co.file as file",
			"co.id as operator_id",
			"co.website as website",
			"co.operator_tags as operator_tags",
		).
		From("current_operators_in_feed coif").
		Join("current_feeds on current_feeds.id = coif.feed_id").
		LeftJoin("current_operators co on co.id = coif.operator_id").
		Where(sq.Eq{"current_feeds.deleted_at": nil}).
		Where(sq.Eq{"co.deleted_at": nil}). // not present, or present but not deleted
		OrderBy("coif.resolved_onestop_id, coif.operator_id")

	if where != nil {
		if where.Merged != nil && !*where.Merged {
			distinct = false
		}

		if where.FeedOnestopID != nil {
			q = q.Where(sq.Eq{"current_feeds.onestop_id": where.FeedOnestopID})
		}

		if where.AgencyID != nil {
			q = q.Where(sq.Eq{"coif.resolved_gtfs_agency_id": where.AgencyID})
		}

		if where.OnestopID != nil {
			q = q.Where(sq.Eq{"coif.resolved_onestop_id": where.OnestopID})
		}

		// Tags
		if where.Tags != nil {
			for _, k := range where.Tags.Keys() {
				if v, ok := where.Tags.Get(k); ok {
					if v == "" {
						q = q.Where("co.operator_tags ?? ?", k)
					} else {
						q = q.Where("co.operator_tags->>? = ?", k, v)
					}
				}
			}
		}

		// Spatial
		if where.Bbox != nil || where.Within != nil || where.Near != nil {
			q = q.
				Join("feed_states fs_geom ON fs_geom.feed_id = coif.feed_id").
				Join("gtfs_agencies a_geom ON a_geom.feed_version_id = fs_geom.feed_version_id AND a_geom.agency_id = coif.resolved_gtfs_agency_id").
				Join("tl_agency_geometries ON tl_agency_geometries.agency_id = a_geom.id")
			if where.Bbox != nil {
				q = q.Where("ST_Intersects(tl_agency_geometries.geometry, ST_MakeEnvelope(?,?,?,?,4326))", where.Bbox.MinLon, where.Bbox.MinLat, where.Bbox.MaxLon, where.Bbox.MaxLat)
			}
			if where.Within != nil && where.Within.Valid {
				q = q.Where("ST_Intersects(tl_agency_geometries.geometry, ?)", where.Within)
			}
			if where.Near != nil {
				radius := checkFloat(&where.Near.Radius, 0, 1_000_000)
				q = q.Where("ST_DWithin(tl_agency_geometries.geometry, ST_MakePoint(?,?), ?)", where.Near.Lon, where.Near.Lat, radius)
			}
		}

		// Places
		if where.Adm0Iso != nil || where.Adm1Iso != nil || where.Adm0Name != nil || where.Adm1Name != nil || where.CityName != nil {
			q = q.
				Join("feed_states ON feed_states.feed_id = coif.feed_id").
				Join("gtfs_agencies ON gtfs_agencies.feed_version_id = feed_states.feed_version_id AND gtfs_agencies.agency_id = coif.resolved_gtfs_agency_id").
				Join("tl_agency_places tlap ON tlap.agency_id = gtfs_agencies.id").
				Join("ne_10m_admin_1_states_provinces ne_admin on ne_admin.name = tlap.adm1name and ne_admin.admin = tlap.adm0name")
			if where.Adm0Iso != nil {
				q = q.Where(sq.ILike{"ne_admin.iso_a2": *where.Adm0Iso})
			}
			if where.Adm1Iso != nil {
				q = q.Where(sq.ILike{"ne_admin.iso_3166_2": *where.Adm1Iso})
			}
			if where.Adm0Name != nil {
				q = q.Where(sq.ILike{"tlap.adm0name": *where.Adm0Name})
			}
			if where.Adm1Name != nil {
				q = q.Where(sq.ILike{"tlap.adm1name": *where.Adm1Name})
			}
			if where.CityName != nil {
				q = q.Where(sq.ILike{"tlap.name": *where.CityName})
			}
		}

		// Handle license filtering
		q = licenseFilter(where.License, q)

		// Text search
		if where.Search != nil && len(*where.Search) > 1 {
			rank, wc := tsTableQuery("coif", *where.Search)
			q = q.Column(rank).Where(sq.Or{
				wc,
				sq.ILike{"coif.resolved_onestop_id": fmt.Sprintf("%%%s%%", *where.Search)},
			})
		}
	}
	if distinct {
		q = q.Distinct().Options("on (coif.resolved_onestop_id)")
	}
	return q
}

func operatorsByAgencyID(_ *int, _ *model.Cursor, agencyIds []int) sq.SelectBuilder {
	q := operatorSelectBase(false, nil)
	q = q.
		Column("a.id as agency_id").
		Join("feed_states fs on fs.feed_id = current_feeds.id").
		Join("gtfs_agencies a on a.feed_version_id = fs.feed_version_id and a.agency_id = coif.resolved_gtfs_agency_id").
		Where(In("a.id", agencyIds))
	return q
}

func operatorSelect(limit *int, after *model.Cursor, ids []int, permFilter *model.PermFilter, where *model.OperatorFilter) sq.SelectBuilder {
	q := operatorSelectBase(true, where)
	if len(ids) > 0 {
		q = q.Where(In("coif.id", ids))
	}

	// Handle permissions
	q = pfJoinCheck(q, permFilter)

	// Outer query - support pagination
	qView := sq.StatementBuilder.Select("t.*").FromSelect(q, "t").OrderBy("id").Limit(checkLimit(limit))
	if after != nil && after.Valid && after.ID > 0 {
		qView = qView.Where(sq.Gt{"t.id": after.ID})
	}
	return qView
}
