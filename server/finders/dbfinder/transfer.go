package dbfinder

import (
	"context"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	sq "github.com/irees/squirrel"
)

func (f *Finder) TransfersByFeedVersionIDs(ctx context.Context, limit *int, after *model.Cursor, where *model.TransferFilter, keys []int) ([][]*model.Transfer, error) {
	var ents []*model.Transfer
	q := lateralWrap(transferSelect(limit, after, where), "feed_versions", "id", "gtfs_transfers", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.Transfer) int { return ent.FeedVersionID }), err
}

func transferSelect(limit *int, after *model.Cursor, where *model.TransferFilter) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_transfers.id",
		"gtfs_transfers.feed_version_id",
		"gtfs_transfers.from_stop_id",
		"gtfs_transfers.to_stop_id",
		"gtfs_transfers.from_route_id",
		"gtfs_transfers.to_route_id",
		"gtfs_transfers.from_trip_id",
		"gtfs_transfers.to_trip_id",
		"gtfs_transfers.transfer_type",
		"gtfs_transfers.min_transfer_time",
		"feed_versions.sha1 AS feed_version_sha1",
		"current_feeds.onestop_id AS feed_onestop_id",
	).From("gtfs_transfers").
		Join("feed_versions ON feed_versions.id = gtfs_transfers.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id")
	if after != nil && after.Valid && after.ID > 0 {
		q = q.Where(sq.Gt{"gtfs_transfers.id": after.ID})
	}
	if where != nil {
		if len(where.Ids) > 0 {
			q = q.Where(In("gtfs_transfers.id", where.Ids))
		}
		if where.TransferType != nil {
			q = q.Where(sq.Eq{"gtfs_transfers.transfer_type": *where.TransferType})
		}
	}
	return joinImported(q).OrderBy("gtfs_transfers.id ASC").Limit(finderCheckLimit(limit))
}
