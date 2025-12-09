package dbfinder

import (
	"context"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	sq "github.com/irees/squirrel"
)

func (f *Finder) BookingRulesByFeedVersionIDs(ctx context.Context, limit *int, where *model.BookingRuleFilter, keys []int) ([][]*model.BookingRule, error) {
	var ents []*model.BookingRule
	q := bookingRuleSelect(limit, nil, nil, where).Where(In("gtfs_booking_rules.feed_version_id", keys))
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.BookingRule) int { return ent.FeedVersionID }), err
}

func (f *Finder) BookingRulesByIDs(ctx context.Context, ids []int) ([]*model.BookingRule, []error) {
	var ents []*model.BookingRule
	q := bookingRuleSelect(nil, nil, ids, nil)
	if err := dbutil.Select(ctx, f.db, q, &ents); err != nil {
		return nil, []error{err}
	}
	return arrangeBy(ids, ents, func(ent *model.BookingRule) int { return ent.ID }), nil
}

func bookingRuleSelect(limit *int, _ *model.Cursor, ids []int, where *model.BookingRuleFilter) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_booking_rules.id",
		"gtfs_booking_rules.feed_version_id",
		"gtfs_booking_rules.created_at",
		"gtfs_booking_rules.updated_at",
		"gtfs_booking_rules.booking_rule_id",
		"gtfs_booking_rules.booking_type",
		"gtfs_booking_rules.prior_notice_duration_min",
		"gtfs_booking_rules.prior_notice_duration_max",
		"gtfs_booking_rules.prior_notice_last_day",
		"gtfs_booking_rules.prior_notice_last_time",
		"gtfs_booking_rules.prior_notice_start_day",
		"gtfs_booking_rules.prior_notice_start_time",
		"gtfs_booking_rules.prior_notice_service_id",
		"gtfs_booking_rules.message",
		"gtfs_booking_rules.pickup_message",
		"gtfs_booking_rules.drop_off_message",
		"gtfs_booking_rules.phone_number",
		"gtfs_booking_rules.info_url",
		"gtfs_booking_rules.booking_url",
		"feed_versions.sha1 AS feed_version_sha1",
		"current_feeds.onestop_id AS feed_onestop_id",
	).From("gtfs_booking_rules").
		Join("feed_versions ON feed_versions.id = gtfs_booking_rules.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id")

	if len(ids) > 0 {
		q = q.Where(In("gtfs_booking_rules.id", ids))
	}
	if where != nil {
		if len(where.Ids) > 0 {
			q = q.Where(In("gtfs_booking_rules.id", where.Ids))
		}
		if where.BookingRuleID != nil && *where.BookingRuleID != "" {
			q = q.Where(sq.Eq{"gtfs_booking_rules.booking_rule_id": *where.BookingRuleID})
		}
	}
	q = q.OrderBy("gtfs_booking_rules.id ASC")
	q = q.Limit(finderCheckLimit(limit))
	return q
}
