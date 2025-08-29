package dbfinder

import (
	"context"
	"fmt"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	sq "github.com/irees/squirrel"
)

func (f *Finder) FindFeeds(ctx context.Context, limit *int, after *model.Cursor, ids []int, where *model.FeedFilter) ([]*model.Feed, error) {
	var ents []*model.Feed
	if err := dbutil.Select(ctx, f.db, feedSelect(limit, after, ids, f.PermFilter(ctx), where), &ents); err != nil {
		return nil, logErr(ctx, err)
	}
	return ents, nil
}

func (f *Finder) FeedsByIDs(ctx context.Context, ids []int) ([]*model.Feed, []error) {
	ents, err := f.FindFeeds(ctx, nil, nil, ids, nil)
	if err != nil {
		return nil, logExtendErr(ctx, len(ids), err)
	}
	return arrangeBy(ids, ents, func(ent *model.Feed) int { return ent.ID }), nil
}

func (f *Finder) FeedStatesByFeedIDs(ctx context.Context, ids []int) ([]*model.FeedState, []error) {
	var ents []*model.FeedState
	err := dbutil.Select(ctx,
		f.db,
		quickSelect("feed_states", nil, nil, nil).Where(In("feed_id", ids)),
		&ents,
	)
	if err != nil {
		return nil, logExtendErr(ctx, len(ids), err)
	}
	return arrangeBy(ids, ents, func(ent *model.FeedState) int { return ent.FeedID }), nil
}

func (f *Finder) FeedFetchesByFeedIDs(ctx context.Context, limit *int, where *model.FeedFetchFilter, keys []int) ([][]*model.FeedFetch, error) {
	q := sq.StatementBuilder.
		Select("*").
		From("feed_fetches").
		Limit(checkLimit(limit)).
		OrderBy("feed_fetches.fetched_at desc")
	if where != nil {
		if where.Success != nil {
			q = q.Where(sq.Eq{"success": *where.Success})
		}
	}
	var ents []*model.FeedFetch
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(q, "current_feeds", "id", "feed_fetches", "feed_id", keys),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.FeedFetch) int { return ent.FeedID }), err
}

func (f *Finder) FeedsByOperatorOnestopIDs(ctx context.Context, limit *int, where *model.FeedFilter, keys []string) ([][]*model.Feed, error) {
	q := feedSelect(nil, nil, nil, f.PermFilter(ctx), where).
		Distinct().Options("on (coif.resolved_onestop_id, current_feeds.id)").
		Column("coif.resolved_onestop_id as with_operator_onestop_id").
		Join("current_operators_in_feed coif on coif.feed_id = current_feeds.id").
		Where(In("coif.resolved_onestop_id", keys))
	var ents []*model.Feed
	err := dbutil.Select(ctx,
		f.db,
		q,
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.Feed) string { return ent.WithOperatorOnestopID.Val }), err
}

func feedSelect(limit *int, after *model.Cursor, ids []int, permFilter *model.PermFilter, where *model.FeedFilter) sq.SelectBuilder {
	q := sq.StatementBuilder.
		Select(
			"current_feeds.id",
			"current_feeds.onestop_id",
			"current_feeds.name",
			// "current_feeds.description",
			"current_feeds.file",
			"current_feeds.auth",
			"current_feeds.languages",
			"current_feeds.feed_tags",
			"current_feeds.urls",
			"current_feeds.spec",
			"current_feeds.license",
		).
		From("current_feeds").
		OrderBy("current_feeds.id asc").
		Limit(checkRange(limit, 0, 10_000))

	if where != nil {
		if where.OnestopID != nil {
			q = q.Where(sq.Eq{"onestop_id": *where.OnestopID})
		}

		if len(where.Spec) > 0 {
			var specs []string
			for _, s := range where.Spec {
				specs = append(specs, s.ToDBString())
			}
			q = q.Where(In("spec", specs))
		}

		// Spatial
		if where.Bbox != nil || where.Within != nil || where.Near != nil {
			q = q.
				Join("feed_states fs_geom on fs_geom.feed_id = current_feeds.id").
				Join("tl_feed_version_geometries fv_geoms on fv_geoms.feed_version_id = fs_geom.feed_version_id")
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

		// Tags
		if where.Tags != nil {
			for _, k := range where.Tags.Keys() {
				if v, ok := where.Tags.Get(k); ok {
					if v == "" {
						q = q.Where("feed_tags ?? ?", k)
					} else {
						q = q.Where("feed_tags->>? = ?", k, v)
					}
				}
			}
		}

		// Fetch error
		if v := where.FetchError; v == nil {
			// nothing
		} else if *v {
			q = q.JoinClause("join lateral (select success from feed_fetches where feed_fetches.feed_id = current_feeds.id order by fetched_at desc limit 1) ff on true").Where(sq.Eq{"ff.success": false})
		} else if !*v {
			q = q.JoinClause("join lateral (select success from feed_fetches where feed_fetches.feed_id = current_feeds.id order by fetched_at desc limit 1) ff on true").Where(sq.Eq{"ff.success": true})
		}

		// Import import status
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
				log.Error().Str("value", v.String()).Msg("unknown imnport status enum")
			}
			// Check the import status of the most recently fetched feed version
			q = q.
				JoinClause("join (select distinct on(fv.feed_id) fv.feed_id, fvgi.in_progress, fvgi.success from feed_version_gtfs_imports fvgi join feed_versions fv on fv.id = fvgi.feed_version_id order by fv.feed_id,fv.fetched_at desc) fvicheck on fvicheck.feed_id = current_feeds.id").
				Where(sq.Eq{"fvicheck.success": checkSuccess}, sq.Eq{"fvicheck.in_progress": checkInProgress})
		}

		// Source URL
		if where.SourceURL != nil {
			urlType := "static_current"
			if where.SourceURL.Type != nil {
				urlType = where.SourceURL.Type.String()
			}
			if where.SourceURL.URL == nil {
				q = q.Where("urls->>? is not null", urlType)
			} else if v := where.SourceURL.CaseSensitive; v != nil && *v {
				q = q.Where("urls->>? = ?", urlType, where.SourceURL.URL)
			} else {
				q = q.Where("lower(urls->>?) = lower(?)", urlType, where.SourceURL.URL)
			}
		}

		// Handle license filtering
		q = licenseFilterTable(where.License, q)

		// Text search
		if where.Search != nil && len(*where.Search) > 1 {
			rank, wc := tsTableQuery("current_feeds", *where.Search)
			q = q.Column(rank).Where(sq.Or{
				wc,
				sq.ILike{"onestop_id": fmt.Sprintf("%%%s%%", *where.Search)},
			})
		}

	}
	if len(ids) > 0 {
		q = q.Where(In("current_feeds.id", ids))
	}
	if after != nil && after.Valid && after.ID > 0 {
		q = q.Where(sq.Gt{"current_feeds.id": after.ID})
	}

	// Handle permissions
	q = pfJoinCheck(q, permFilter)
	return q
}
