package dbfinder

import (
	"context"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	sq "github.com/irees/squirrel"
)

func (f *Finder) BookingRulesByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.BookingRule, error) {
	var ents []*model.BookingRule
	q := bookingRuleSelect(limit, nil, nil).Where(In("gtfs_booking_rules.feed_version_id", keys))
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.BookingRule) int { return ent.FeedVersionID }), err
}

func (f *Finder) BookingRulesByIDs(ctx context.Context, ids []int) ([]*model.BookingRule, []error) {
	var ents []*model.BookingRule
	q := bookingRuleSelect(nil, nil, ids)
	if err := dbutil.Select(ctx, f.db, q, &ents); err != nil {
		return nil, []error{err}
	}
	return arrangeBy(ids, ents, func(ent *model.BookingRule) int { return ent.ID }), nil
}

func bookingRuleSelect(limit *int, after *model.Cursor, ids []int) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_booking_rules.*",
		"feed_versions.sha1 AS feed_version_sha1",
		"feeds.onestop_id AS feed_onestop_id",
	).From("gtfs_booking_rules").
		Join("feed_versions ON feed_versions.id = gtfs_booking_rules.feed_version_id").
		Join("feeds ON feeds.id = feed_versions.feed_id")

	if len(ids) > 0 {
		q = q.Where(In("gtfs_booking_rules.id", ids))
	}
	q = q.Limit(finderCheckLimit(limit))
	return q
}
