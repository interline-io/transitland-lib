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
	return fareEntitySelect("gtfs_fare_attributes", limit, ids,
		"fare_id",
		"price",
		"currency_type",
		"payment_method",
		"transfers",
		"agency_id",
		"transfer_duration",
	)
}

func (f *Finder) FareRulesByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.FareRule, error) {
	var ents []*model.FareRule
	q := lateralWrap(fareRuleSelect(limit, nil), "feed_versions", "id", "gtfs_fare_rules", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.FareRule) int { return ent.FeedVersionID }), err
}

func fareRuleSelect(limit *int, ids []int) sq.SelectBuilder {
	return fareEntitySelect("gtfs_fare_rules", limit, ids,
		"fare_id",
		"route_id",
		"origin_id",
		"destination_id",
		"contains_id",
	)
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
	return fareEntitySelect("gtfs_fare_media", limit, ids,
		"fare_media_id",
		"fare_media_name",
		"fare_media_type",
	)
}

func (f *Finder) FareProductsByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.FareProduct, error) {
	var ents []*model.FareProduct
	q := lateralWrap(fareProductSelect(limit, nil), "feed_versions", "id", "gtfs_fare_products", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.FareProduct) int { return ent.FeedVersionID }), err
}

func fareProductSelect(limit *int, ids []int) sq.SelectBuilder {
	return fareEntitySelect("gtfs_fare_products", limit, ids,
		"fare_product_id",
		"fare_product_name",
		"amount",
		"currency",
		"rider_category_id",
		"fare_media_id",
		"duration_start",
		"duration_amount",
		"duration_unit",
		"duration_type",
	)
}

func (f *Finder) FareLegRulesByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.FareLegRule, error) {
	var ents []*model.FareLegRule
	q := lateralWrap(fareLegRuleSelect(limit, nil), "feed_versions", "id", "gtfs_fare_leg_rules", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.FareLegRule) int { return ent.FeedVersionID }), err
}

func fareLegRuleSelect(limit *int, ids []int) sq.SelectBuilder {
	return fareEntitySelect("gtfs_fare_leg_rules", limit, ids,
		"leg_group_id",
		"network_id",
		"from_area_id",
		"to_area_id",
		"from_timeframe_group_id",
		"to_timeframe_group_id",
		"fare_product_id",
		"rule_priority",
		"transfer_only",
	)
}

func (f *Finder) FareLegJoinRulesByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.FareLegJoinRule, error) {
	var ents []*model.FareLegJoinRule
	q := lateralWrap(fareLegJoinRuleSelect(limit, nil), "feed_versions", "id", "gtfs_fare_leg_join_rules", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.FareLegJoinRule) int { return ent.FeedVersionID }), err
}

func fareLegJoinRuleSelect(limit *int, ids []int) sq.SelectBuilder {
	return fareEntitySelect("gtfs_fare_leg_join_rules", limit, ids,
		"from_network_id",
		"to_network_id",
		"from_stop_id",
		"to_stop_id",
	)
}

func (f *Finder) FareTransferRulesByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.FareTransferRule, error) {
	var ents []*model.FareTransferRule
	q := lateralWrap(fareTransferRuleSelect(limit, nil), "feed_versions", "id", "gtfs_fare_transfer_rules", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.FareTransferRule) int { return ent.FeedVersionID }), err
}

func fareTransferRuleSelect(limit *int, ids []int) sq.SelectBuilder {
	return fareEntitySelect("gtfs_fare_transfer_rules", limit, ids,
		"from_leg_group_id",
		"to_leg_group_id",
		"transfer_count",
		"duration_limit",
		"duration_limit_type",
		"fare_transfer_type",
		"fare_product_id",
		"filter_fare_product_id",
	)
}

func (f *Finder) RiderCategoriesByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.RiderCategory, error) {
	var ents []*model.RiderCategory
	q := lateralWrap(riderCategorySelect(limit, nil), "feed_versions", "id", "gtfs_rider_categories", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.RiderCategory) int { return ent.FeedVersionID }), err
}

func riderCategorySelect(limit *int, ids []int) sq.SelectBuilder {
	return fareEntitySelect("gtfs_rider_categories", limit, ids,
		"rider_category_id",
		"rider_category_name",
		"is_default_fare_category",
		"eligibility_url",
		"min_age",
		"max_age",
	)
}

func (f *Finder) TimeframesByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.Timeframe, error) {
	var ents []*model.Timeframe
	q := lateralWrap(timeframeSelect(limit, nil), "feed_versions", "id", "gtfs_timeframes", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.Timeframe) int { return ent.FeedVersionID }), err
}

func timeframeSelect(limit *int, ids []int) sq.SelectBuilder {
	return fareEntitySelect("gtfs_timeframes", limit, ids,
		"timeframe_group_id",
		"start_time",
		"end_time",
		"service_id",
	)
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
	return fareEntitySelect("gtfs_areas", limit, ids,
		"area_id",
		"area_name",
	)
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
	return fareEntitySelect("gtfs_networks", limit, ids,
		"network_id",
		"network_name",
	)
}

func (f *Finder) RouteNetworksByFeedVersionIDs(ctx context.Context, limit *int, keys []int) ([][]*model.RouteNetwork, error) {
	var ents []*model.RouteNetwork
	q := lateralWrap(routeNetworkSelect(limit, nil), "feed_versions", "id", "gtfs_route_networks", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.RouteNetwork) int { return ent.FeedVersionID }), err
}

func routeNetworkSelect(limit *int, ids []int) sq.SelectBuilder {
	return fareEntitySelect("gtfs_route_networks", limit, ids,
		"network_id",
		"route_id",
	)
}

func (f *Finder) StopAreasByAreaIDs(ctx context.Context, limit *int, keys []int) ([][]*model.StopArea, error) {
	var ents []*model.StopArea
	q := lateralWrap(stopAreaSelect(limit, nil), "gtfs_areas", "id", "gtfs_stop_areas", "area_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.StopArea) int { return ent.AreaID.Int() }), err
}

func stopAreaSelect(limit *int, ids []int) sq.SelectBuilder {
	return fareEntitySelect("gtfs_stop_areas", limit, ids,
		"area_id",
		"stop_id",
	)
}

// fareEntitySelect selects the listed columns of a fares table, plus the id,
// feed version, and feed metadata common to every entity.
func fareEntitySelect(table string, limit *int, ids []int, cols ...string) sq.SelectBuilder {
	table = az09(table)
	sel := []string{table + ".id", table + ".feed_version_id"}
	for _, col := range cols {
		sel = append(sel, table+"."+az09(col))
	}
	sel = append(sel,
		"feed_versions.sha1 AS feed_version_sha1",
		"current_feeds.onestop_id AS feed_onestop_id",
	)
	q := sq.StatementBuilder.Select(sel...).
		From(table).
		Join("feed_versions ON feed_versions.id = " + table + ".feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id")
	if len(ids) > 0 {
		q = q.Where(In(table+".id", ids))
	}
	return q.OrderBy(table + ".id ASC").Limit(finderCheckLimit(limit))
}
