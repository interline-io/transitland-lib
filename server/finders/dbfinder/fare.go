package dbfinder

import (
	"context"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	sq "github.com/irees/squirrel"
)

// GTFS Fares v1 and v2 entities. These tables have no filters of their own:
// each is loaded as a child of its feed version (or, for stop_areas, its area),
// with the limit applied per parent.
//
// The importer rewrites some text references (fare_leg_rules.network_id,
// from_area_id, to_area_id, and every fare_leg_join_rules column) to the
// referenced row's internal id when that row exists, and leaves them as GTFS
// ids otherwise: a network named only in routes.network_id has no
// gtfs_networks row. Those selects join the referenced table on a numeric
// stored value and return its GTFS id, falling back to the stored value.
// The importer leaves a reference unresolved only when the referenced table
// has no row for it (routes.network_id) or when reference errors are allowed,
// so a numeric GTFS id matching an unrelated internal id needs both.

func (f *Finder) FareAttributesByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.FareAttribute, error) {
	var ents []*model.FareAttribute
	q := lateralWrap(fareAttributeSelect(limit, nil), "feed_versions", "id", "gtfs_fare_attributes", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.FareAttribute) int { return ent.FeedVersionID }), err
}

func (f *Finder) FareAttributesByIDs(ctx context.Context, ids []int) ([]*model.FareAttribute, []error) {
	var ents []*model.FareAttribute
	q := fareAttributeSelect(nil, ids)
	if err := dbutil.Select(ctx, f.db, q, &ents); err != nil {
		return nil, []error{err}
	}
	return arrangeBy(ids, ents, func(ent *model.FareAttribute) int { return ent.ID }), nil
}

func fareAttributeSelect(limit *int, ids []int) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_fare_attributes.id",
		"gtfs_fare_attributes.feed_version_id",
		"gtfs_fare_attributes.fare_id",
		"gtfs_fare_attributes.price",
		"gtfs_fare_attributes.currency_type",
		"gtfs_fare_attributes.payment_method",
		"gtfs_fare_attributes.transfers",
		"gtfs_fare_attributes.agency_id",
		"gtfs_fare_attributes.transfer_duration",
		"feed_versions.sha1 AS feed_version_sha1",
		"current_feeds.onestop_id AS feed_onestop_id",
	).From("gtfs_fare_attributes").
		Join("feed_versions ON feed_versions.id = gtfs_fare_attributes.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id")
	if len(ids) > 0 {
		q = q.Where(In("gtfs_fare_attributes.id", ids))
	}
	return q.OrderBy("gtfs_fare_attributes.id ASC").Limit(finderCheckLimit(limit))
}

func (f *Finder) FareRulesByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.FareRule, error) {
	var ents []*model.FareRule
	q := lateralWrap(fareRuleSelect(limit, nil), "feed_versions", "id", "gtfs_fare_rules", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.FareRule) int { return ent.FeedVersionID }), err
}

func fareRuleSelect(limit *int, ids []int) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_fare_rules.id",
		"gtfs_fare_rules.feed_version_id",
		"gtfs_fare_rules.fare_id",
		"gtfs_fare_rules.route_id",
		"gtfs_fare_rules.origin_id",
		"gtfs_fare_rules.destination_id",
		"gtfs_fare_rules.contains_id",
		"feed_versions.sha1 AS feed_version_sha1",
		"current_feeds.onestop_id AS feed_onestop_id",
	).From("gtfs_fare_rules").
		Join("feed_versions ON feed_versions.id = gtfs_fare_rules.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id")
	if len(ids) > 0 {
		q = q.Where(In("gtfs_fare_rules.id", ids))
	}
	return q.OrderBy("gtfs_fare_rules.id ASC").Limit(finderCheckLimit(limit))
}

func (f *Finder) FareMediaByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.FareMedia, error) {
	var ents []*model.FareMedia
	q := lateralWrap(fareMediaSelect(limit, nil), "feed_versions", "id", "gtfs_fare_media", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.FareMedia) int { return ent.FeedVersionID }), err
}

func (f *Finder) FareMediaByIDs(ctx context.Context, ids []int) ([]*model.FareMedia, []error) {
	var ents []*model.FareMedia
	q := fareMediaSelect(nil, ids)
	if err := dbutil.Select(ctx, f.db, q, &ents); err != nil {
		return nil, []error{err}
	}
	return arrangeBy(ids, ents, func(ent *model.FareMedia) int { return ent.ID }), nil
}

func fareMediaSelect(limit *int, ids []int) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_fare_media.id",
		"gtfs_fare_media.feed_version_id",
		"gtfs_fare_media.fare_media_id",
		"gtfs_fare_media.fare_media_name",
		"gtfs_fare_media.fare_media_type",
		"feed_versions.sha1 AS feed_version_sha1",
		"current_feeds.onestop_id AS feed_onestop_id",
	).From("gtfs_fare_media").
		Join("feed_versions ON feed_versions.id = gtfs_fare_media.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id")
	if len(ids) > 0 {
		q = q.Where(In("gtfs_fare_media.id", ids))
	}
	return q.OrderBy("gtfs_fare_media.id ASC").Limit(finderCheckLimit(limit))
}

func (f *Finder) FareProductsByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.FareProduct, error) {
	var ents []*model.FareProduct
	q := lateralWrap(fareProductSelect(limit, nil), "feed_versions", "id", "gtfs_fare_products", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.FareProduct) int { return ent.FeedVersionID }), err
}

func fareProductSelect(limit *int, ids []int) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_fare_products.id",
		"gtfs_fare_products.feed_version_id",
		"gtfs_fare_products.fare_product_id",
		"gtfs_fare_products.fare_product_name",
		"gtfs_fare_products.amount",
		"gtfs_fare_products.currency",
		"gtfs_fare_products.rider_category_id",
		"gtfs_fare_products.fare_media_id",
		"gtfs_fare_products.duration_start",
		"gtfs_fare_products.duration_amount",
		"gtfs_fare_products.duration_unit",
		"gtfs_fare_products.duration_type",
		"feed_versions.sha1 AS feed_version_sha1",
		"current_feeds.onestop_id AS feed_onestop_id",
	).From("gtfs_fare_products").
		Join("feed_versions ON feed_versions.id = gtfs_fare_products.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id")
	if len(ids) > 0 {
		q = q.Where(In("gtfs_fare_products.id", ids))
	}
	return q.OrderBy("gtfs_fare_products.id ASC").Limit(finderCheckLimit(limit))
}

func (f *Finder) FareLegRulesByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.FareLegRule, error) {
	var ents []*model.FareLegRule
	q := lateralWrap(fareLegRuleSelect(limit, nil), "feed_versions", "id", "gtfs_fare_leg_rules", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.FareLegRule) int { return ent.FeedVersionID }), err
}

func fareLegRuleSelect(limit *int, ids []int) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_fare_leg_rules.id",
		"gtfs_fare_leg_rules.feed_version_id",
		"gtfs_fare_leg_rules.leg_group_id",
		"gtfs_fare_leg_rules.from_timeframe_group_id",
		"gtfs_fare_leg_rules.to_timeframe_group_id",
		"gtfs_fare_leg_rules.fare_product_id",
		"gtfs_fare_leg_rules.rule_priority",
		"gtfs_fare_leg_rules.transfer_only",
		"COALESCE(ref_network.network_id, gtfs_fare_leg_rules.network_id) AS network_id",
		"COALESCE(ref_from_area.area_id, gtfs_fare_leg_rules.from_area_id) AS from_area_id",
		"COALESCE(ref_to_area.area_id, gtfs_fare_leg_rules.to_area_id) AS to_area_id",
		"feed_versions.sha1 AS feed_version_sha1",
		"current_feeds.onestop_id AS feed_onestop_id",
	).From("gtfs_fare_leg_rules").
		Join("feed_versions ON feed_versions.id = gtfs_fare_leg_rules.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id").
		LeftJoin("gtfs_networks ref_network ON ref_network.feed_version_id = gtfs_fare_leg_rules.feed_version_id AND ref_network.id = (CASE WHEN gtfs_fare_leg_rules.network_id ~ '^[0-9]{1,18}$' THEN gtfs_fare_leg_rules.network_id::bigint END)").
		LeftJoin("gtfs_areas ref_from_area ON ref_from_area.feed_version_id = gtfs_fare_leg_rules.feed_version_id AND ref_from_area.id = (CASE WHEN gtfs_fare_leg_rules.from_area_id ~ '^[0-9]{1,18}$' THEN gtfs_fare_leg_rules.from_area_id::bigint END)").
		LeftJoin("gtfs_areas ref_to_area ON ref_to_area.feed_version_id = gtfs_fare_leg_rules.feed_version_id AND ref_to_area.id = (CASE WHEN gtfs_fare_leg_rules.to_area_id ~ '^[0-9]{1,18}$' THEN gtfs_fare_leg_rules.to_area_id::bigint END)")
	if len(ids) > 0 {
		q = q.Where(In("gtfs_fare_leg_rules.id", ids))
	}
	return q.OrderBy("gtfs_fare_leg_rules.id ASC").Limit(finderCheckLimit(limit))
}

func (f *Finder) FareLegJoinRulesByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.FareLegJoinRule, error) {
	var ents []*model.FareLegJoinRule
	q := lateralWrap(fareLegJoinRuleSelect(limit, nil), "feed_versions", "id", "gtfs_fare_leg_join_rules", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.FareLegJoinRule) int { return ent.FeedVersionID }), err
}

func fareLegJoinRuleSelect(limit *int, ids []int) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_fare_leg_join_rules.id",
		"gtfs_fare_leg_join_rules.feed_version_id",
		"COALESCE(ref_from_network.network_id, gtfs_fare_leg_join_rules.from_network_id) AS from_network_id",
		"COALESCE(ref_to_network.network_id, gtfs_fare_leg_join_rules.to_network_id) AS to_network_id",
		"COALESCE(ref_from_stop.stop_id, gtfs_fare_leg_join_rules.from_stop_id) AS from_stop_id",
		"COALESCE(ref_to_stop.stop_id, gtfs_fare_leg_join_rules.to_stop_id) AS to_stop_id",
		"feed_versions.sha1 AS feed_version_sha1",
		"current_feeds.onestop_id AS feed_onestop_id",
	).From("gtfs_fare_leg_join_rules").
		Join("feed_versions ON feed_versions.id = gtfs_fare_leg_join_rules.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id").
		LeftJoin("gtfs_networks ref_from_network ON ref_from_network.feed_version_id = gtfs_fare_leg_join_rules.feed_version_id AND ref_from_network.id = (CASE WHEN gtfs_fare_leg_join_rules.from_network_id ~ '^[0-9]{1,18}$' THEN gtfs_fare_leg_join_rules.from_network_id::bigint END)").
		LeftJoin("gtfs_networks ref_to_network ON ref_to_network.feed_version_id = gtfs_fare_leg_join_rules.feed_version_id AND ref_to_network.id = (CASE WHEN gtfs_fare_leg_join_rules.to_network_id ~ '^[0-9]{1,18}$' THEN gtfs_fare_leg_join_rules.to_network_id::bigint END)").
		LeftJoin("gtfs_stops ref_from_stop ON ref_from_stop.feed_version_id = gtfs_fare_leg_join_rules.feed_version_id AND ref_from_stop.id = (CASE WHEN gtfs_fare_leg_join_rules.from_stop_id ~ '^[0-9]{1,18}$' THEN gtfs_fare_leg_join_rules.from_stop_id::bigint END)").
		LeftJoin("gtfs_stops ref_to_stop ON ref_to_stop.feed_version_id = gtfs_fare_leg_join_rules.feed_version_id AND ref_to_stop.id = (CASE WHEN gtfs_fare_leg_join_rules.to_stop_id ~ '^[0-9]{1,18}$' THEN gtfs_fare_leg_join_rules.to_stop_id::bigint END)")
	if len(ids) > 0 {
		q = q.Where(In("gtfs_fare_leg_join_rules.id", ids))
	}
	return q.OrderBy("gtfs_fare_leg_join_rules.id ASC").Limit(finderCheckLimit(limit))
}

func (f *Finder) FareTransferRulesByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.FareTransferRule, error) {
	var ents []*model.FareTransferRule
	q := lateralWrap(fareTransferRuleSelect(limit, nil), "feed_versions", "id", "gtfs_fare_transfer_rules", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.FareTransferRule) int { return ent.FeedVersionID }), err
}

func fareTransferRuleSelect(limit *int, ids []int) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_fare_transfer_rules.id",
		"gtfs_fare_transfer_rules.feed_version_id",
		"gtfs_fare_transfer_rules.from_leg_group_id",
		"gtfs_fare_transfer_rules.to_leg_group_id",
		"gtfs_fare_transfer_rules.transfer_count",
		"gtfs_fare_transfer_rules.duration_limit",
		"gtfs_fare_transfer_rules.duration_limit_type",
		"gtfs_fare_transfer_rules.fare_transfer_type",
		"gtfs_fare_transfer_rules.fare_product_id",
		"gtfs_fare_transfer_rules.filter_fare_product_id",
		"feed_versions.sha1 AS feed_version_sha1",
		"current_feeds.onestop_id AS feed_onestop_id",
	).From("gtfs_fare_transfer_rules").
		Join("feed_versions ON feed_versions.id = gtfs_fare_transfer_rules.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id")
	if len(ids) > 0 {
		q = q.Where(In("gtfs_fare_transfer_rules.id", ids))
	}
	return q.OrderBy("gtfs_fare_transfer_rules.id ASC").Limit(finderCheckLimit(limit))
}

func (f *Finder) RiderCategoriesByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.RiderCategory, error) {
	var ents []*model.RiderCategory
	q := lateralWrap(riderCategorySelect(limit, nil), "feed_versions", "id", "gtfs_rider_categories", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.RiderCategory) int { return ent.FeedVersionID }), err
}

func riderCategorySelect(limit *int, ids []int) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_rider_categories.id",
		"gtfs_rider_categories.feed_version_id",
		"gtfs_rider_categories.rider_category_id",
		"gtfs_rider_categories.rider_category_name",
		"gtfs_rider_categories.is_default_fare_category",
		"gtfs_rider_categories.eligibility_url",
		"gtfs_rider_categories.min_age",
		"gtfs_rider_categories.max_age",
		"feed_versions.sha1 AS feed_version_sha1",
		"current_feeds.onestop_id AS feed_onestop_id",
	).From("gtfs_rider_categories").
		Join("feed_versions ON feed_versions.id = gtfs_rider_categories.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id")
	if len(ids) > 0 {
		q = q.Where(In("gtfs_rider_categories.id", ids))
	}
	return q.OrderBy("gtfs_rider_categories.id ASC").Limit(finderCheckLimit(limit))
}

func (f *Finder) TimeframesByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.Timeframe, error) {
	var ents []*model.Timeframe
	q := lateralWrap(timeframeSelect(limit, nil), "feed_versions", "id", "gtfs_timeframes", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.Timeframe) int { return ent.FeedVersionID }), err
}

func timeframeSelect(limit *int, ids []int) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_timeframes.id",
		"gtfs_timeframes.feed_version_id",
		"gtfs_timeframes.timeframe_group_id",
		"gtfs_timeframes.start_time",
		"gtfs_timeframes.end_time",
		"gtfs_timeframes.service_id",
		"feed_versions.sha1 AS feed_version_sha1",
		"current_feeds.onestop_id AS feed_onestop_id",
	).From("gtfs_timeframes").
		Join("feed_versions ON feed_versions.id = gtfs_timeframes.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id")
	if len(ids) > 0 {
		q = q.Where(In("gtfs_timeframes.id", ids))
	}
	return q.OrderBy("gtfs_timeframes.id ASC").Limit(finderCheckLimit(limit))
}

func (f *Finder) AreasByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.Area, error) {
	var ents []*model.Area
	q := lateralWrap(areaSelect(limit, nil), "feed_versions", "id", "gtfs_areas", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.Area) int { return ent.FeedVersionID }), err
}

func (f *Finder) AreasByIDs(ctx context.Context, ids []int) ([]*model.Area, []error) {
	var ents []*model.Area
	q := areaSelect(nil, ids)
	if err := dbutil.Select(ctx, f.db, q, &ents); err != nil {
		return nil, []error{err}
	}
	return arrangeBy(ids, ents, func(ent *model.Area) int { return ent.ID }), nil
}

func areaSelect(limit *int, ids []int) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_areas.id",
		"gtfs_areas.feed_version_id",
		"gtfs_areas.area_id",
		"gtfs_areas.area_name",
		"feed_versions.sha1 AS feed_version_sha1",
		"current_feeds.onestop_id AS feed_onestop_id",
	).From("gtfs_areas").
		Join("feed_versions ON feed_versions.id = gtfs_areas.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id")
	if len(ids) > 0 {
		q = q.Where(In("gtfs_areas.id", ids))
	}
	return q.OrderBy("gtfs_areas.id ASC").Limit(finderCheckLimit(limit))
}

func (f *Finder) NetworksByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.Network, error) {
	var ents []*model.Network
	q := lateralWrap(networkSelect(limit, nil), "feed_versions", "id", "gtfs_networks", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.Network) int { return ent.FeedVersionID }), err
}

func (f *Finder) NetworksByIDs(ctx context.Context, ids []int) ([]*model.Network, []error) {
	var ents []*model.Network
	q := networkSelect(nil, ids)
	if err := dbutil.Select(ctx, f.db, q, &ents); err != nil {
		return nil, []error{err}
	}
	return arrangeBy(ids, ents, func(ent *model.Network) int { return ent.ID }), nil
}

func networkSelect(limit *int, ids []int) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_networks.id",
		"gtfs_networks.feed_version_id",
		"gtfs_networks.network_id",
		"gtfs_networks.network_name",
		"feed_versions.sha1 AS feed_version_sha1",
		"current_feeds.onestop_id AS feed_onestop_id",
	).From("gtfs_networks").
		Join("feed_versions ON feed_versions.id = gtfs_networks.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id")
	if len(ids) > 0 {
		q = q.Where(In("gtfs_networks.id", ids))
	}
	return q.OrderBy("gtfs_networks.id ASC").Limit(finderCheckLimit(limit))
}

func (f *Finder) RouteNetworksByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.RouteNetwork, error) {
	var ents []*model.RouteNetwork
	q := lateralWrap(routeNetworkSelect(limit, nil), "feed_versions", "id", "gtfs_route_networks", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.RouteNetwork) int { return ent.FeedVersionID }), err
}

func routeNetworkSelect(limit *int, ids []int) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_route_networks.id",
		"gtfs_route_networks.feed_version_id",
		"gtfs_route_networks.network_id",
		"gtfs_route_networks.route_id",
		"feed_versions.sha1 AS feed_version_sha1",
		"current_feeds.onestop_id AS feed_onestop_id",
	).From("gtfs_route_networks").
		Join("feed_versions ON feed_versions.id = gtfs_route_networks.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id")
	if len(ids) > 0 {
		q = q.Where(In("gtfs_route_networks.id", ids))
	}
	return q.OrderBy("gtfs_route_networks.id ASC").Limit(finderCheckLimit(limit))
}

func (f *Finder) StopAreasByAreaIDs(ctx context.Context, limit *int, keys []int) ([][]*model.StopArea, error) {
	var ents []*model.StopArea
	q := lateralWrap(stopAreaSelect(limit, nil), "gtfs_areas", "id", "gtfs_stop_areas", "area_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.StopArea) int { return ent.AreaID.Int() }), err
}

func stopAreaSelect(limit *int, ids []int) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_stop_areas.id",
		"gtfs_stop_areas.feed_version_id",
		"gtfs_stop_areas.area_id",
		"gtfs_stop_areas.stop_id",
		"feed_versions.sha1 AS feed_version_sha1",
		"current_feeds.onestop_id AS feed_onestop_id",
	).From("gtfs_stop_areas").
		Join("feed_versions ON feed_versions.id = gtfs_stop_areas.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id")
	if len(ids) > 0 {
		q = q.Where(In("gtfs_stop_areas.id", ids))
	}
	return q.OrderBy("gtfs_stop_areas.id ASC").Limit(finderCheckLimit(limit))
}
