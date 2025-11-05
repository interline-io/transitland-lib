package dbfinder

import (
	"context"
	"errors"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tt"
	sq "github.com/irees/squirrel"
)

func (f *Finder) FindFeedVersions(ctx context.Context, limit *int, after *model.Cursor, ids []int, where *model.FeedVersionFilter) ([]*model.FeedVersion, error) {
	var ents []*model.FeedVersion
	if err := dbutil.Select(ctx, f.db, feedVersionSelect(limit, after, ids, f.PermFilter(ctx), where), &ents); err != nil {
		return nil, logErr(ctx, err)
	}
	return ents, nil
}

func (f *Finder) FeedVersionsByIDs(ctx context.Context, ids []int) ([]*model.FeedVersion, []error) {
	ents, err := f.FindFeedVersions(ctx, nil, nil, ids, nil)
	if err != nil {
		return nil, logExtendErr(ctx, len(ids), err)
	}
	return arrangeBy(ids, ents, func(ent *model.FeedVersion) int { return ent.ID }), nil
}

func (f *Finder) FeedVersionGtfsImportByFeedVersionIDs(ctx context.Context, ids []int) ([]*model.FeedVersionGtfsImport, []error) {
	var ents []*model.FeedVersionGtfsImport
	err := dbutil.Select(ctx,
		f.db,
		quickSelect("feed_version_gtfs_imports", nil, nil, nil).Where(In("feed_version_id", ids)),
		&ents,
	)
	if err != nil {
		return nil, logExtendErr(ctx, len(ids), err)
	}
	return arrangeBy(ids, ents, func(ent *model.FeedVersionGtfsImport) int { return ent.FeedVersionID }), nil
}

func (f *Finder) FeedVersionServiceWindowByFeedVersionIDs(ctx context.Context, ids []int) ([]*model.FeedVersionServiceWindow, []error) {
	var ents []*model.FeedVersionServiceWindow
	err := dbutil.Select(ctx,
		f.db,
		quickSelect("feed_version_service_windows", nil, nil, nil).Where(In("feed_version_id", ids)),
		&ents,
	)
	if err != nil {
		return nil, logExtendErr(ctx, len(ids), err)
	}
	return arrangeBy(ids, ents, func(ent *model.FeedVersionServiceWindow) int { return ent.FeedVersionID }), nil
}

func (f *Finder) FeedVersionGeometryByIDs(ctx context.Context, ids []int) ([]*tt.Polygon, []error) {
	if len(ids) == 0 {
		return nil, nil
	}
	qents := []*feedVersionGeometry{}
	if err := dbutil.Select(ctx, f.db, feedVersionGeometrySelect(ids), &qents); err != nil {
		return nil, logExtendErr(ctx, len(ids), err)
	}
	group := map[int]*tt.Polygon{}
	for _, ent := range qents {
		group[ent.FeedVersionID] = ent.Geometry
	}
	ents := make([]*tt.Polygon, len(ids))
	for i, id := range ids {
		ents[i] = group[id]
	}
	return ents, nil
}

func (f *Finder) FeedVersionFileInfosByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.FeedVersionFileInfo, error) {
	var ents []*model.FeedVersionFileInfo
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			quickSelectOrder("feed_version_file_infos", limit, nil, nil, "feed_version_id"),
			"feed_versions",
			"id",
			"feed_version_file_infos",
			"feed_version_id",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.FeedVersionFileInfo) int { return ent.FeedVersionID }), err
}

func (f *Finder) FeedVersionsByFeedIDs(ctx context.Context, limit *int, where *model.FeedVersionFilter, keys []int) ([][]*model.FeedVersion, error) {
	var ents []*model.FeedVersion
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			feedVersionSelect(limit, nil, nil, f.PermFilter(ctx), where),
			"current_feeds",
			"id",
			"feed_versions",
			"feed_id",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.FeedVersion) int { return ent.FeedID }), err
}

func (f *Finder) FeedVersionServiceLevelsByFeedVersionIDs(ctx context.Context, limit *int, where *model.FeedVersionServiceLevelFilter, keys []int) ([][]*model.FeedVersionServiceLevel, error) {
	var ents []*model.FeedVersionServiceLevel
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			feedVersionServiceLevelSelect(limit, nil, nil, f.PermFilter(ctx), where),
			"feed_versions",
			"id",
			"feed_version_service_levels",
			"feed_version_id",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.FeedVersionServiceLevel) int { return ent.FeedVersionID }), err
}

func (f *Finder) FindFeedVersionServiceWindow(ctx context.Context, fvid int) (*model.ServiceWindow, error) {
	a, _, err := f.fvslCache.Get(ctx, fvid)
	if err != nil || a == nil || a.Location == nil {
		return nil, errors.New("no service window found")
	}
	// Get local time
	nowLocal := time.Now().In(a.Location)
	if model.ForContext(ctx).Clock != nil {
		nowLocal = model.ForContext(ctx).Clock.Now().In(a.Location)
	}
	// Copy back to model
	ret := &model.ServiceWindow{
		NowLocal:     nowLocal,
		StartDate:    a.StartDate,
		EndDate:      a.EndDate,
		FallbackWeek: a.FallbackWeek,
	}
	return ret, err
}

func (f *Finder) FeedInfosByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.FeedInfo, error) {
	var ents []*model.FeedInfo
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			quickSelectOrder("gtfs_feed_infos", limit, nil, nil, "feed_version_id"),
			"feed_versions",
			"id",
			"gtfs_feed_infos",
			"feed_version_id",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.FeedInfo) int { return ent.FeedVersionID }), err
}

func feedVersionSelect(limit *int, after *model.Cursor, ids []int, permFilter *model.PermFilter, where *model.FeedVersionFilter) sq.SelectBuilder {
	q := sq.StatementBuilder.
		Select(
			"feed_versions.id",
			"feed_versions.feed_id",
			"feed_versions.sha1",
			"feed_versions.fetched_at",
			"feed_versions.url",
			"feed_versions.earliest_calendar_date",
			"feed_versions.latest_calendar_date",
			"feed_versions.created_by",
			"feed_versions.updated_by",
			"feed_versions.name",
			"feed_versions.description",
			"feed_versions.file",
		).
		From("feed_versions").
		Join("current_feeds on current_feeds.id = feed_versions.feed_id").
		Limit(finderCheckLimit(limit)).
		OrderBy("feed_versions.fetched_at desc, feed_versions.id desc")

	if where != nil {
		if where.Sha1 != nil {
			q = q.Where(sq.Eq{"feed_versions.sha1": *where.Sha1})
		}
		if where.File != nil {
			q = q.Where(sq.Eq{"feed_versions.file": where.File})
		}
		if len(where.FeedIds) > 0 {
			q = q.Where(sq.Eq{"feed_versions.feed_id": where.FeedIds})
		}
		if where.FeedOnestopID != nil {
			q = q.Where(sq.Eq{"current_feeds.onestop_id": *where.FeedOnestopID})
		}
		if len(where.Ids) > 0 {
			ids = append(ids, where.Ids...)
		}

		// Spatial
		if where.Bbox != nil || where.Within != nil || where.Near != nil {
			q = q.Join("tl_feed_version_geometries fv_geoms on fv_geoms.feed_version_id = feed_versions.id")
			if where.Bbox != nil {
				q = q.Where("ST_Intersects(fv_geoms.geometry, ST_MakeEnvelope(?,?,?,?,4326))", where.Bbox.MinLon, where.Bbox.MinLat, where.Bbox.MaxLon, where.Bbox.MaxLat)
			}
			if where.Within != nil && where.Within.Valid {
				q = q.Where("ST_Intersects(fv_geoms.geometry, ?)", where.Within)
			}
			if where.Near != nil {
				radius := checkFloat(&where.Near.Radius, 0, 1_000_000)
				q = q.Where("ST_DWithin(fv_geoms.geometry, ST_MakePoint(?,?), ?)", where.Near.Lon, where.Near.Lat, radius)
			}
		}

		// Coverage
		if covers := where.Covers; covers != nil {
			// Handle fetched at
			if covers.FetchedBefore != nil {
				q = q.Where(sq.Lt{"feed_versions.fetched_at": covers.FetchedBefore})
			}
			if covers.FetchedAfter != nil {
				q = q.Where(sq.Gt{"feed_versions.fetched_at": covers.FetchedAfter})
			}

			// Handle flexible service extent
			joinFvsw := false
			if covers.StartDate != nil && covers.StartDate.Valid {
				joinFvsw = true
				q = q.
					Where(sq.LtOrEq{"greatest(fvsw.feed_start_date,fvsw.earliest_calendar_date)": covers.StartDate.Val}).
					Where(sq.GtOrEq{"least(fvsw.feed_end_date,fvsw.latest_calendar_date)": covers.StartDate.Val})
			}
			if covers.EndDate != nil && covers.EndDate.Valid {
				joinFvsw = true
				q = q.
					Where(sq.LtOrEq{"greatest(fvsw.feed_start_date,fvsw.earliest_calendar_date)": covers.EndDate.Val}).
					Where(sq.GtOrEq{"least(fvsw.feed_end_date,fvsw.latest_calendar_date)": covers.EndDate.Val})
			}

			// Handle feed_info.txt extent
			if covers.FeedStartDate != nil && covers.FeedStartDate.Valid {
				joinFvsw = true
				q = q.
					Where(sq.LtOrEq{"fvsw.feed_start_date": covers.FeedStartDate.Val}).
					Where(sq.GtOrEq{"fvsw.feed_end_date": covers.FeedStartDate.Val})
			}
			if covers.FeedEndDate != nil && covers.FeedEndDate.Valid {
				joinFvsw = true
				q = q.
					Where(sq.LtOrEq{"fvsw.feed_start_date": covers.FeedEndDate.Val}).
					Where(sq.GtOrEq{"fvsw.feed_end_date": covers.FeedEndDate.Val})
			}

			// Handle service extent
			if covers.EarliestCalendarDate != nil && covers.EarliestCalendarDate.Valid {
				q = q.
					Where(sq.LtOrEq{"feed_versions.earliest_calendar_date": covers.EarliestCalendarDate.Val}).
					Where(sq.GtOrEq{"feed_versions.latest_calendar_date": covers.EarliestCalendarDate.Val})
			}
			if covers.LatestCalendarDate != nil && covers.LatestCalendarDate.Valid {
				q = q.
					Where(sq.LtOrEq{"feed_versions.earliest_calendar_date": covers.LatestCalendarDate.Val}).
					Where(sq.GtOrEq{"feed_versions.latest_calendar_date": covers.LatestCalendarDate.Val})
			}

			// Add feed version service windows
			if joinFvsw {
				q = q.Join("feed_version_service_windows fvsw on fvsw.feed_version_id = feed_versions.id")
			}
		}

		// Import import status
		// Similar logic to FeedSelect
		if where.ImportStatus != nil {
			// in_progress must be false to check success and vice-versa
			var checkSuccess bool
			var checkInProgress bool
			switch v := *where.ImportStatus; v {
			case model.ImportStatusSuccess:
				checkSuccess = true
				checkInProgress = false
			case model.ImportStatusInProgress:
				checkSuccess = false
				checkInProgress = true
			case model.ImportStatusError:
				checkSuccess = false
				checkInProgress = false
			default:
				log.Error().Str("value", v.String()).Msg("unknown import status enum")
			}
			q = q.Join(`feed_version_gtfs_imports fvgi on fvgi.feed_version_id = feed_versions.id`).
				Where(sq.Eq{"fvgi.success": checkSuccess}, sq.Eq{"fvgi.in_progress": checkInProgress})
		}

		// Handle license filtering
		q = licenseFilter(where.License, q)
	}
	if len(ids) > 0 {
		q = q.Where(In("feed_versions.id", ids))
	}
	if after != nil && after.Valid && after.ID > 0 {
		q = q.Where(sq.Expr("(feed_versions.fetched_at,feed_versions.id) < (select fetched_at,id from feed_versions where id = ?)", after.ID))
	}

	// Handle permissions
	q = pfJoinCheckFv(q, permFilter)
	return q
}

func feedVersionServiceLevelSelect(limit *int, after *model.Cursor, ids []int, _ *model.PermFilter, where *model.FeedVersionServiceLevelFilter) sq.SelectBuilder {
	q := sq.StatementBuilder.
		Select(
			"feed_version_service_levels.id",
			"feed_version_service_levels.feed_version_id",
			"feed_version_service_levels.route_id",
			"feed_version_service_levels.start_date",
			"feed_version_service_levels.end_date",
			"feed_version_service_levels.monday",
			"feed_version_service_levels.tuesday",
			"feed_version_service_levels.wednesday",
			"feed_version_service_levels.thursday",
			"feed_version_service_levels.friday",
			"feed_version_service_levels.saturday",
			"feed_version_service_levels.sunday",
		).
		From("feed_version_service_levels").
		Limit(finderCheckLimit(limit)).
		OrderBy("feed_version_service_levels.id")

	q = q.Where(sq.Eq{"route_id": nil})
	if where != nil {
		if where.StartDate != nil {
			q = q.Where(sq.LtOrEq{"start_date": where.StartDate})
		}
		if where.EndDate != nil {
			q = q.Where(sq.GtOrEq{"end_date": where.EndDate})
		}
	}
	if len(ids) > 0 {
		q = q.Where(In("feed_version_service_levels.id", ids))
	}
	if after != nil && after.Valid && after.ID > 0 {
		q = q.Where(sq.Gt{"feed_version_service_levels.id": after.ID})
	}
	return q
}

type feedVersionGeometry struct {
	FeedVersionID int
	Geometry      *tt.Polygon
}

func feedVersionGeometrySelect(ids []int) sq.SelectBuilder {
	return sq.StatementBuilder.
		Select("feed_version_id", "geometry").
		From("tl_feed_version_geometries").
		Where(In("feed_version_id", ids))
}
