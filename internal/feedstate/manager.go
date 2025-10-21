package feedstate

import (
	"context"
	"fmt"
	"sort"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/tldb"
	sq "github.com/irees/squirrel"
)

// FeedVersionInfo contains metadata about a feed version
type FeedVersionInfo struct {
	FeedVersionID int
	FeedID        int
	OnestopID     string
	FeedName      string
}

// Manager handles feed state and materialized table operations
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

	currentFeedVersionID, feedIsActive := feedStates[feedID]

	// If this feed version is already active, do nothing
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

	// Set in feed_states
	_, err = m.adapter.DBX().ExecContext(ctx, "UPDATE feed_states SET feed_version_id = $1 WHERE feed_id = $2", feedVersionID, feedID)
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

	currentFeedVersionID, feedIsActive := feedStates[feedID]

	// If this feed version is not active, do nothing
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

	// Unset in feed_states
	_, err = m.adapter.DBX().ExecContext(ctx, "UPDATE feed_states SET feed_version_id = NULL WHERE feed_id = $1", feedID)
	if err != nil {
		return fmt.Errorf("failed to unset feed version in feed_states: %w", err)
	}

	// Remove from materialized tables
	if err := m.DematerializeFeedVersion(ctx, feedID); err != nil {
		return fmt.Errorf("failed to remove from materialized tables: %w", err)
	}

	return nil
}

// GetFeedIDForFeedVersion gets the feed_id for a given feed_version_id (public version)
func (m *Manager) GetFeedIDForFeedVersion(ctx context.Context, fvid int) (int, error) {
	var feedID int
	err := m.adapter.Get(ctx, &feedID, "SELECT feed_id FROM feed_versions WHERE id = $1", fvid)
	return feedID, err
}

// getActiveFeedStates returns a map of feed_id to feed_version_id for all active feeds
func (m *Manager) getActiveFeedStates(ctx context.Context) (map[int]int, error) {
	rows, err := m.adapter.DBX().QueryxContext(ctx, "SELECT feed_id, feed_version_id FROM feed_states WHERE feed_version_id IS NOT NULL")
	if err != nil {
		return nil, fmt.Errorf("failed to query feed_states: %w", err)
	}
	defer rows.Close()

	feedStates := make(map[int]int)
	for rows.Next() {
		var feedID, feedVersionID int
		if err := rows.Scan(&feedID, &feedVersionID); err != nil {
			return nil, fmt.Errorf("failed to scan feed_states row: %w", err)
		}
		feedStates[feedID] = feedVersionID
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating feed_states rows: %w", err)
	}

	return feedStates, nil
}

// DematerializeFeedVersion removes all routes/stops/agencies for a feed from materialized tables
func (m *Manager) DematerializeFeedVersion(ctx context.Context, fvid int) error {
	// Remove routes
	_, err := m.adapter.Sqrl().
		Delete("tl_materialized_active_routes").
		Where(sq.Eq{"feed_version_id": fvid}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove routes for feed version %d: %w", fvid, err)
	}

	// Remove stops
	_, err = m.adapter.Sqrl().
		Delete("tl_materialized_active_stops").
		Where(sq.Eq{"feed_version_id": fvid}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove stops for feed version %d: %w", fvid, err)
	}

	// Remove agencies
	_, err = m.adapter.Sqrl().
		Delete("tl_materialized_active_agencies").
		Where(sq.Eq{"feed_version_id": fvid}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove agencies for feed version %d: %w", fvid, err)
	}

	return nil
}

// MaterializeFeedVersion inserts routes/stops/agencies for a feed version into materialized tables
func (m *Manager) MaterializeFeedVersion(ctx context.Context, fvid int) error {
	// Insert routes with derived data
	routeInsert := `
	INSERT INTO tl_materialized_active_routes (
		id, 
		route_id, 
		agency_id, 
		agency_name,
		route_short_name, 
		route_long_name, 
		route_desc, 
		route_type, 
		route_url, 
		route_color, 
		route_text_color,
		feed_version_id, 
		feed_id, 
		onestop_id, 
		geometry, 
		textsearch
	)
	SELECT 
		gtfs_routes.id, 
		gtfs_routes.route_id, 
		gtfs_agencies.agency_id, 
		gtfs_agencies.agency_name,
		gtfs_routes.route_short_name, 
		gtfs_routes.route_long_name,
		gtfs_routes.route_desc, 
		gtfs_routes.route_type, 
		gtfs_routes.route_url, 
		gtfs_routes.route_color, 
		gtfs_routes.route_text_color,
		feed_versions.id as feed_version_id,
		feed_Versions.feed_id,
		osid.onestop_id, 
		tl_route_geometries.geometry,
		gtfs_routes.textsearch
	FROM gtfs_routes
	JOIN gtfs_agencies ON gtfs_agencies.id = gtfs_routes.agency_id
	JOIN feed_versions ON feed_versions.id = gtfs_routes.feed_version_id
	LEFT JOIN feed_version_route_onestop_ids osid ON osid.entity_id = gtfs_routes.route_id and osid.feed_version_id = feed_versions.id
	LEFT JOIN tl_route_geometries ON tl_route_geometries.route_id = gtfs_routes.id
	WHERE gtfs_routes.feed_version_id = $1
	`

	_, err := m.adapter.DBX().ExecContext(ctx, routeInsert, fvid)
	if err != nil {
		return fmt.Errorf("failed to insert routes for feed version %d: %w", fvid, err)
	}

	// Insert stops with derived data
	stopInsert := `
	INSERT INTO tl_materialized_active_stops (
		id, 
		stop_id, 
		stop_code, 
		stop_name, 
		stop_desc, 
		stop_url, 
		location_type, 
		parent_station, 
		feed_version_id, 
		feed_id,
		onestop_id, 
		geometry, 
		textsearch
	)
	SELECT 
		gtfs_stops.id, 
		gtfs_stops.stop_id, 
		gtfs_stops.stop_code, 
		gtfs_stops.stop_name, 
		gtfs_stops.stop_desc, 
		gtfs_stops.stop_url, 
		gtfs_stops.location_type,
		gtfs_stops.parent_station, 
		feed_versions.id as feed_version_id,
		feed_versions.feed_id,
		osid.onestop_id, 
		gtfs_stops.geometry, 
		gtfs_stops.textsearch
	FROM gtfs_stops
	JOIN feed_versions ON feed_versions.id = gtfs_stops.feed_version_id
	LEFT JOIN feed_version_stop_onestop_ids osid ON osid.entity_id = gtfs_stops.stop_id and osid.feed_version_id = feed_versions.id
	WHERE gtfs_stops.feed_version_id = $1
	`

	_, err = m.adapter.DBX().ExecContext(ctx, stopInsert, fvid)
	if err != nil {
		return fmt.Errorf("failed to insert stops for feed version %d: %w", fvid, err)
	}

	// Insert agencies with derived data
	agencyInsert := `
	INSERT INTO tl_materialized_active_agencies (
		id, 
		agency_id, 
		agency_name, 
		agency_url, 
		agency_timezone, 
		agency_lang, 
		agency_phone, 
		agency_fare_url, 
		agency_email,
		feed_version_id, 
		feed_id,
		onestop_id, 
		geometry, 
		textsearch
	)
	SELECT 
		gtfs_agencies.id, 
		gtfs_agencies.agency_id, 
		gtfs_agencies.agency_name, 
		gtfs_agencies.agency_url, 
		gtfs_agencies.agency_timezone, 
		gtfs_agencies.agency_lang, 
		gtfs_agencies.agency_phone, 
		gtfs_agencies.agency_fare_url, 
		gtfs_agencies.agency_email,
		feed_versions.id as feed_version_id,
		feed_versions.feed_id,
		osid.onestop_id, 
		tl_agency_geometries.geometry, 
		gtfs_agencies.textsearch
	FROM gtfs_agencies
	JOIN feed_versions ON feed_versions.id = gtfs_agencies.feed_version_id
	LEFT JOIN feed_version_agency_onestop_ids osid ON osid.entity_id = gtfs_agencies.agency_id and osid.feed_version_id = feed_versions.id
	LEFT JOIN tl_agency_geometries ON tl_agency_geometries.agency_id = gtfs_agencies.id
	WHERE gtfs_agencies.feed_version_id = $1
	`

	_, err = m.adapter.DBX().ExecContext(ctx, agencyInsert, fvid)
	if err != nil {
		return fmt.Errorf("failed to insert agencies for feed version %d: %w", fvid, err)
	}

	return nil
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
// NOTE: This method does NOT handle transactions - the caller must manage transactions
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
