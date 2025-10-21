package feedstate

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sort"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/tldb"
	sq "github.com/irees/squirrel"
)

// Manager handles feed state and materialized table operations
// NOTE: Methods do NOT handle transactions - the caller must manage transactions
type Manager struct {
	adapter tldb.Adapter
}

// NewManager creates a new feed state manager
func NewManager(adapter tldb.Adapter) *Manager {
	return &Manager{
		adapter: adapter,
	}
}

// ActivateFeedVersion activates a feed version by setting it in feed_states and adding to materialized tables
// If another version of the same feed is currently active, it will be deactivated first
func (m *Manager) ActivateFeedVersion(ctx context.Context, feedVersionID int) error {
	// Get the feed_id for this feed version
	feedID, err := m.GetFeedIDForFeedVersion(ctx, feedVersionID)
	if err != nil {
		return fmt.Errorf("failed to get feed_id for feed version %d: %w", feedVersionID, err)
	}

	// Get current feed states to check if this feed version is already active
	feedStates, err := m.getActiveFeedStates(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active feed states: %w", err)
	}

	// If this feed version is already active, do nothing
	currentFeedVersionID, feedIsActive := feedStates[feedID]
	if feedIsActive && currentFeedVersionID == feedVersionID {
		log.For(ctx).Info().
			Int("feed_version_id", feedVersionID).
			Int("feed_id", feedID).
			Msg("Feed version is already active")
		return nil
	}

	// If another version is active, deactivate it first
	if feedIsActive {
		log.For(ctx).Info().
			Int("old_feed_version_id", currentFeedVersionID).
			Int("new_feed_version_id", feedVersionID).
			Int("feed_id", feedID).
			Msg("Deactivating current feed version before activating new one")
		if err := m.DeactivateFeedVersion(ctx, currentFeedVersionID); err != nil {
			return fmt.Errorf("failed to deactivate current feed version %d: %w", currentFeedVersionID, err)
		}
	}

	// Activate the new feed version
	log.For(ctx).Info().
		Int("feed_version_id", feedVersionID).
		Int("feed_id", feedID).
		Msg("Activating feed version")

	// Set in feed_states using Squirrel
	_, err = m.adapter.Sqrl().
		Update("feed_states").
		Set("feed_version_id", feedVersionID).
		Where(sq.Eq{"feed_id": feedID}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to set feed version in feed_states: %w", err)
	}

	// Add to materialized tables
	if err := m.MaterializeFeedVersion(ctx, feedVersionID); err != nil {
		return fmt.Errorf("failed to add to materialized tables: %w", err)
	}

	return nil
}

// DeactivateFeedVersion deactivates a feed version by removing it from feed_states and materialized tables
// If the feed version is not currently active, does nothing
func (m *Manager) DeactivateFeedVersion(ctx context.Context, feedVersionID int) error {
	// Get the feed_id for this feed version
	feedID, err := m.GetFeedIDForFeedVersion(ctx, feedVersionID)
	if err != nil {
		return fmt.Errorf("failed to get feed_id for feed version %d: %w", feedVersionID, err)
	}

	// Get current feed states to check if this feed version is active
	feedStates, err := m.getActiveFeedStates(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active feed states: %w", err)
	}

	// If this feed version is not active, do nothing
	currentFeedVersionID, feedIsActive := feedStates[feedID]
	if !feedIsActive || currentFeedVersionID != feedVersionID {
		log.For(ctx).Info().
			Int("feed_version_id", feedVersionID).
			Int("feed_id", feedID).
			Msg("Feed version is not currently active")
		return nil
	}

	log.For(ctx).Info().
		Int("feed_version_id", feedVersionID).
		Int("feed_id", feedID).
		Msg("Deactivating feed version")

	// Unset in feed_states using Squirrel
	_, err = m.adapter.Sqrl().
		Update("feed_states").
		Set("feed_version_id", nil).
		Where(sq.Eq{"feed_id": feedID}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to unset feed version in feed_states: %w", err)
	}

	// Remove from materialized tables
	if err := m.DematerializeFeedVersion(ctx, feedVersionID); err != nil {
		return fmt.Errorf("failed to remove from materialized tables: %w", err)
	}

	return nil
}

// GetFeedIDForFeedVersion gets the feed_id for a given feed_version_id (public version)
func (m *Manager) GetFeedIDForFeedVersion(ctx context.Context, fvid int) (int, error) {
	var feedID int
	err := dbutil.Get(ctx, m.adapter.DBX(), m.adapter.Sqrl().
		Select("feed_id").
		From("feed_versions").
		Where(sq.Eq{"id": fvid}), &feedID)
	return feedID, err
}

// getActiveFeedStates returns a map of feed_id to feed_version_id for all active feeds
func (m *Manager) getActiveFeedStates(ctx context.Context) (map[int]int, error) {
	type feedState struct {
		FeedID        int `db:"feed_id"`
		FeedVersionID int `db:"feed_version_id"`
	}

	var states []feedState
	err := dbutil.Select(ctx, m.adapter.DBX(), m.adapter.Sqrl().
		Select("feed_id", "feed_version_id").
		From("feed_states").
		Where("feed_version_id IS NOT NULL"), &states)
	if err != nil {
		return nil, fmt.Errorf("failed to query feed_states: %w", err)
	}

	feedStates := make(map[int]int)
	for _, state := range states {
		feedStates[state.FeedID] = state.FeedVersionID
	}

	return feedStates, nil
}

// DematerializeFeedVersion removes all routes/stops/agencies for a feed from materialized tables
func (m *Manager) DematerializeFeedVersion(ctx context.Context, feedVersionID int) error {
	// Remove routes
	_, err := m.adapter.Sqrl().
		Delete("tl_materialized_active_routes").
		Where(sq.Eq{"feed_version_id": feedVersionID}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove routes for feed version %d: %w", feedVersionID, err)
	}

	// Remove stops
	_, err = m.adapter.Sqrl().
		Delete("tl_materialized_active_stops").
		Where(sq.Eq{"feed_version_id": feedVersionID}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove stops for feed version %d: %w", feedVersionID, err)
	}

	// Remove agencies
	_, err = m.adapter.Sqrl().
		Delete("tl_materialized_active_agencies").
		Where(sq.Eq{"feed_version_id": feedVersionID}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove agencies for feed version %d: %w", feedVersionID, err)
	}

	return nil
}

// MaterializeFeedVersion inserts routes/stops/agencies for a feed version into materialized tables
func (m *Manager) MaterializeFeedVersion(ctx context.Context, feedVersionID int) error {
	// Insert routes with derived data using Squirrel
	routeQuery := m.adapter.Sqrl().
		Insert("tl_materialized_active_routes").
		Columns(
			"id",
			"route_id",
			"route_short_name",
			"route_long_name",
			"route_desc",
			"route_type",
			"route_url",
			"route_color",
			"route_text_color",
			"route_sort_order",
			"network_id",
			"as_route",
			"continuous_pickup",
			"continuous_drop_off",
			"agency_id",
			"gtfs_agency_id",
			"agency_name",
			"feed_version_id",
			"feed_id",
			"onestop_id",
			"textsearch",
		).
		Select(m.adapter.Sqrl().
			Select(
				"gtfs_routes.id",
				"gtfs_routes.route_id",
				"gtfs_routes.route_short_name",
				"gtfs_routes.route_long_name",
				"gtfs_routes.route_desc",
				"gtfs_routes.route_type",
				"gtfs_routes.route_url",
				"gtfs_routes.route_color",
				"gtfs_routes.route_text_color",
				"gtfs_routes.route_sort_order",
				"gtfs_routes.network_id",
				"gtfs_routes.as_route",
				"gtfs_routes.continuous_pickup",
				"gtfs_routes.continuous_drop_off",
				"gtfs_routes.agency_id",
				"gtfs_agencies.agency_id as gtfs_agency_id",
				"gtfs_agencies.agency_name",
				"feed_versions.id as feed_version_id",
				"feed_versions.feed_id",
				"osid.onestop_id",
				"gtfs_routes.textsearch",
			).
			From("gtfs_routes").
			Join("gtfs_agencies ON gtfs_agencies.id = gtfs_routes.agency_id").
			Join("feed_versions ON feed_versions.id = gtfs_routes.feed_version_id").
			LeftJoin("feed_version_route_onestop_ids osid ON osid.entity_id = gtfs_routes.route_id AND osid.feed_version_id = feed_versions.id").
			Where(sq.Eq{"gtfs_routes.feed_version_id": feedVersionID}))

	_, err := routeQuery.ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert routes for feed version %d: %w", feedVersionID, err)
	}

	// Insert stops with derived data using Squirrel
	stopQuery := m.adapter.Sqrl().
		Insert("tl_materialized_active_stops").
		Columns(
			"id",
			"stop_id",
			"stop_code",
			"stop_name",
			"stop_desc",
			"zone_id",
			"stop_url",
			"location_type",
			"stop_timezone",
			"wheelchair_boarding",
			"parent_station",
			"level_id",
			"area_id",
			"tts_stop_name",
			"platform_code",
			"feed_version_id",
			"feed_id",
			"onestop_id",
			"geometry",
			"textsearch",
		).
		Select(m.adapter.Sqrl().
			Select(
				"gtfs_stops.id",
				"gtfs_stops.stop_id",
				"gtfs_stops.stop_code",
				"gtfs_stops.stop_name",
				"gtfs_stops.stop_desc",
				"gtfs_stops.zone_id",
				"gtfs_stops.stop_url",
				"gtfs_stops.location_type",
				"gtfs_stops.stop_timezone",
				"gtfs_stops.wheelchair_boarding",
				"gtfs_stops.parent_station",
				"gtfs_stops.level_id",
				"gtfs_stops.area_id",
				"gtfs_stops.tts_stop_name",
				"gtfs_stops.platform_code",
				"feed_versions.id as feed_version_id",
				"feed_versions.feed_id",
				"osid.onestop_id",
				"gtfs_stops.geometry",
				"gtfs_stops.textsearch",
			).
			From("gtfs_stops").
			Join("feed_versions ON feed_versions.id = gtfs_stops.feed_version_id").
			LeftJoin("feed_version_stop_onestop_ids osid ON osid.entity_id = gtfs_stops.stop_id AND osid.feed_version_id = feed_versions.id").
			Where(sq.Eq{"gtfs_stops.feed_version_id": feedVersionID}))

	_, err = stopQuery.ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert stops for feed version %d: %w", feedVersionID, err)
	}

	// Insert agencies with derived data using Squirrel
	agencyQuery := m.adapter.Sqrl().
		Insert("tl_materialized_active_agencies").
		Columns(
			"id",
			"agency_id",
			"agency_name",
			"agency_url",
			"agency_timezone",
			"agency_lang",
			"agency_phone",
			"agency_fare_url",
			"agency_email",
			"feed_version_id",
			"feed_id",
			"onestop_id",
			"textsearch",
		).
		Select(m.adapter.Sqrl().
			Select(
				"gtfs_agencies.id",
				"gtfs_agencies.agency_id",
				"gtfs_agencies.agency_name",
				"gtfs_agencies.agency_url",
				"gtfs_agencies.agency_timezone",
				"gtfs_agencies.agency_lang",
				"gtfs_agencies.agency_phone",
				"gtfs_agencies.agency_fare_url",
				"gtfs_agencies.agency_email",
				"feed_versions.id as feed_version_id",
				"feed_versions.feed_id",
				"osid.onestop_id",
				"gtfs_agencies.textsearch",
			).
			From("gtfs_agencies").
			Join("feed_versions ON feed_versions.id = gtfs_agencies.feed_version_id").
			LeftJoin("feed_version_agency_onestop_ids osid ON osid.entity_id = gtfs_agencies.agency_id AND osid.feed_version_id = feed_versions.id").
			Where(sq.Eq{"gtfs_agencies.feed_version_id": feedVersionID}))

	_, err = agencyQuery.ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert agencies for feed version %d: %w", feedVersionID, err)
	}

	return nil
}

func (m *Manager) GetMaterializedFeedVersions(ctx context.Context) ([]int, error) {
	// Check all three tables, then dedup
	feedVersionIds := map[int]bool{}
	for _, table := range []string{
		"tl_materialized_active_routes",
		"tl_materialized_active_stops",
		"tl_materialized_active_agencies",
	} {
		var ids []int
		err := dbutil.Select(ctx, m.adapter.DBX(), m.adapter.Sqrl().
			Select("feed_version_id").
			Distinct().Options("on (feed_version_id)").
			From(table), &ids)
		if err != nil {
			return nil, fmt.Errorf("failed to get materialized feed versions from %s: %w", table, err)
		}
		for _, id := range ids {
			feedVersionIds[id] = true
		}
	}
	return slices.Collect(maps.Keys(feedVersionIds)), nil
}

// GetActiveFeedVersions returns a list of currently active feed version IDs
func (m *Manager) GetActiveFeedVersions(ctx context.Context) ([]int, error) {
	feedStates, err := m.getActiveFeedStates(ctx)
	if err != nil {
		return nil, err
	}

	var feedVersionIDs []int
	for _, feedVersionID := range feedStates {
		feedVersionIDs = append(feedVersionIDs, feedVersionID)
	}

	// Sort for consistent ordering
	sort.Ints(feedVersionIDs)
	return feedVersionIDs, nil
}

// SetActiveFeedVersions sets the complete active set of feed versions
// Any currently active feed versions not in the specified set will be deactivated,
// and all feed versions in the specified set will be activated.
func (m *Manager) SetActiveFeedVersions(ctx context.Context, feedVersionIDs []int) error {
	log.For(ctx).Info().
		Ints("target_feed_version_ids", feedVersionIDs).
		Msg("Setting active feed versions")

	// Get currently active feed versions
	currentActive, err := m.GetActiveFeedVersions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current active feed versions: %w", err)
	}

	// Create sets for efficient comparison
	targetSet := make(map[int]bool)
	for _, fvid := range feedVersionIDs {
		targetSet[fvid] = true
	}

	currentSet := make(map[int]bool)
	for _, fvid := range currentActive {
		currentSet[fvid] = true
	}

	// Deactivate any feed versions that are currently active but not in the target set
	for _, fvid := range currentActive {
		if !targetSet[fvid] {
			log.For(ctx).Info().
				Int("feed_version_id", fvid).
				Msg("Deactivating feed version (not in target set)")

			if err := m.DeactivateFeedVersion(ctx, fvid); err != nil {
				return fmt.Errorf("failed to deactivate feed version %d: %w", fvid, err)
			}
		}
	}

	// Activate all feed versions in the target set
	for _, fvid := range feedVersionIDs {
		if !currentSet[fvid] {
			log.For(ctx).Info().
				Int("feed_version_id", fvid).
				Msg("Activating feed version (new in target set)")
		}

		if err := m.ActivateFeedVersion(ctx, fvid); err != nil {
			return fmt.Errorf("failed to activate feed version %d: %w", fvid, err)
		}
	}

	log.For(ctx).Info().
		Ints("active_feed_version_ids", feedVersionIDs).
		Msg("Successfully set active feed versions")

	return nil
}
