package materialized

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
	sq "github.com/irees/squirrel"
)

// RefreshPlan contains the changes needed to update materialized tables
type RefreshPlan struct {
	FeedsToAdd    []FeedVersionChange
	FeedsToRemove []FeedVersionChange
	FeedsToUpdate []FeedVersionChange
}

// FeedVersionChange represents a change in feed version for a feed
type FeedVersionChange struct {
	FeedID           int
	OldFeedVersionID *int
	NewFeedVersionID int
	OnestopID        string
	FeedName         string
}

// RefreshStats contains statistics about the materialized tables
type RefreshStats struct {
	RouteCount  int        `json:"route_count"`
	StopCount   int        `json:"stop_count"`
	AgencyCount int        `json:"agency_count"`
	FeedCount   int        `json:"feed_count"`
	LastRefresh *time.Time `json:"last_refresh,omitempty"`
	TotalFeeds  int        `json:"total_feeds"`
	OutOfSync   int        `json:"out_of_sync"`
}

// MaterializedIndexState tracks which feed version is materialized for each feed
type MaterializedIndexState struct {
	FeedID                    int       `db:"feed_id" json:"feed_id"`
	MaterializedFeedVersionID int       `db:"materialized_feed_version_id" json:"materialized_feed_version_id"`
	LastMaterializedAt        time.Time `db:"last_materialized_at" json:"last_materialized_at"`
	tt.DatabaseEntity
}

// TableName returns the database table name
func (mis *MaterializedIndexState) TableName() string {
	return "tl_materialized_index_state"
}

// EntityID returns the entity ID as a string
func (mis *MaterializedIndexState) EntityID() string {
	return strconv.Itoa(mis.FeedID)
}

// FeedHealthCheck represents the health status of a feed's materialized data
type FeedHealthCheck struct {
	FeedID               int        `json:"feed_id"`
	OnestopID            string     `json:"onestop_id"`
	FeedName             string     `json:"feed_name"`
	SourceRoutes         int        `json:"source_routes"`
	MaterializedRoutes   int        `json:"materialized_routes"`
	SourceStops          int        `json:"source_stops"`
	MaterializedStops    int        `json:"materialized_stops"`
	SourceAgencies       int        `json:"source_agencies"`
	MaterializedAgencies int        `json:"materialized_agencies"`
	Status               string     `json:"status"`
	LastMaterializedAt   *time.Time `json:"last_materialized_at,omitempty"`
}

// IsHealthy returns true if the materialized data matches the source data
func (fhc *FeedHealthCheck) IsHealthy() bool {
	return fhc.Status == "OK" &&
		fhc.SourceRoutes == fhc.MaterializedRoutes &&
		fhc.SourceStops == fhc.MaterializedStops &&
		fhc.SourceAgencies == fhc.MaterializedAgencies
}

// RefreshOptions contains options for refreshing materialized tables
type RefreshOptions struct {
	FeedIDs     []int
	DryRun      bool
	Force       bool
	Concurrency int
}

// Refresher handles materialized table operations
type Refresher struct {
	adapter tldb.Adapter
}

// NewRefresher creates a new materialized table refresher
func NewRefresher(adapter tldb.Adapter) *Refresher {
	return &Refresher{
		adapter: adapter,
	}
}

// RefreshMaterializedTables performs incremental refresh of materialized tables
func (r *Refresher) RefreshMaterializedTables(ctx context.Context, opts RefreshOptions) error {
	plan, err := r.generateRefreshPlan(ctx, opts.FeedIDs)
	if err != nil {
		return fmt.Errorf("failed to generate refresh plan: %w", err)
	}

	if opts.DryRun {
		log.For(ctx).Info().
			Int("feeds_to_add", len(plan.FeedsToAdd)).
			Int("feeds_to_remove", len(plan.FeedsToRemove)).
			Int("feeds_to_update", len(plan.FeedsToUpdate)).
			Msg("Dry run - would perform these changes")
		return nil
	}

	return r.adapter.Tx(func(atx tldb.Adapter) error {
		// Remove obsolete feed versions
		for _, change := range plan.FeedsToRemove {
			log.For(ctx).Info().
				Int("feed_id", change.FeedID).
				Str("onestop_id", change.OnestopID).
				Msg("Removing feed from materialized tables")
			if err := r.removeFeedVersion(ctx, atx, change); err != nil {
				return fmt.Errorf("failed to remove feed %d: %w", change.FeedID, err)
			}
		}

		// Update changed feed versions
		for _, change := range plan.FeedsToUpdate {
			log.For(ctx).Info().
				Int("feed_id", change.FeedID).
				Str("onestop_id", change.OnestopID).
				Int("old_fv_id", *change.OldFeedVersionID).
				Int("new_fv_id", change.NewFeedVersionID).
				Msg("Updating feed in materialized tables")
			if err := r.updateFeedVersion(ctx, atx, change); err != nil {
				return fmt.Errorf("failed to update feed %d: %w", change.FeedID, err)
			}
		}

		// Add new feed versions
		for _, change := range plan.FeedsToAdd {
			log.For(ctx).Info().
				Int("feed_id", change.FeedID).
				Str("onestop_id", change.OnestopID).
				Int("fv_id", change.NewFeedVersionID).
				Msg("Adding feed to materialized tables")
			if err := r.addFeedVersion(ctx, atx, change); err != nil {
				return fmt.Errorf("failed to add feed %d: %w", change.FeedID, err)
			}
		}

		return nil
	})
}

// RefreshSpecificFeed refreshes materialized tables for a specific feed
func (r *Refresher) RefreshSpecificFeed(ctx context.Context, feedID int) error {
	opts := RefreshOptions{FeedIDs: []int{feedID}}
	return r.RefreshMaterializedTables(ctx, opts)
}

// FullRefresh rebuilds all materialized tables from scratch
func (r *Refresher) FullRefresh(ctx context.Context) error {
	log.For(ctx).Info().Msg("Starting full refresh of materialized tables")

	return r.adapter.Tx(func(atx tldb.Adapter) error {
		// Clear all materialized data
		if err := r.clearAllMaterializedData(ctx, atx); err != nil {
			return fmt.Errorf("failed to clear materialized data: %w", err)
		}

		// Get all active feeds
		plan, err := r.generateRefreshPlan(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to generate refresh plan: %w", err)
		}

		// Add all active feeds
		for _, change := range plan.FeedsToAdd {
			log.For(ctx).Info().
				Int("feed_id", change.FeedID).
				Str("onestop_id", change.OnestopID).
				Msg("Adding feed during full refresh")
			if err := r.addFeedVersion(ctx, atx, change); err != nil {
				return fmt.Errorf("failed to add feed %d during full refresh: %w", change.FeedID, err)
			}
		}

		log.For(ctx).Info().
			Int("feeds_added", len(plan.FeedsToAdd)).
			Msg("Full refresh completed")

		return nil
	})
}

// generateRefreshPlan compares current feed_states with materialized state
func (r *Refresher) generateRefreshPlan(ctx context.Context, feedIDs []int) (*RefreshPlan, error) {
	query := `
	WITH current_active AS (
		SELECT 
			fs.feed_id,
			fs.feed_version_id,
			cf.onestop_id,
			cf.name as feed_name
		FROM feed_states fs
		JOIN current_feeds cf ON cf.id = fs.feed_id
		WHERE fs.feed_version_id IS NOT NULL
		  AND fs.public = true
		  AND cf.deleted_at IS NULL
		  AND ($1::int[] IS NULL OR fs.feed_id = ANY($1))
	),
	materialized_state AS (
		SELECT 
			feed_id,
			materialized_feed_version_id as feed_version_id
		FROM tl_materialized_index_state
		WHERE ($1::int[] IS NULL OR feed_id = ANY($1))
	)
	SELECT 
		COALESCE(ca.feed_id, ms.feed_id) as feed_id,
		ca.feed_version_id as current_fv_id,
		ms.feed_version_id as materialized_fv_id,
		COALESCE(ca.onestop_id, '') as onestop_id,
		COALESCE(ca.feed_name, 'UNKNOWN') as feed_name,
		CASE 
			WHEN ca.feed_id IS NULL THEN 'remove'
			WHEN ms.feed_id IS NULL THEN 'add'
			WHEN ca.feed_version_id != ms.feed_version_id THEN 'update'
			ELSE 'unchanged'
		END as action
	FROM current_active ca
	FULL OUTER JOIN materialized_state ms ON ca.feed_id = ms.feed_id
	WHERE CASE 
		WHEN ca.feed_id IS NULL THEN 'remove'
		WHEN ms.feed_id IS NULL THEN 'add'
		WHEN ca.feed_version_id != ms.feed_version_id THEN 'update'
		ELSE 'unchanged'
	END != 'unchanged'
	`

	var feedIDsParam interface{}
	if len(feedIDs) > 0 {
		feedIDsParam = feedIDs
	}

	rows, err := r.adapter.DBX().QueryxContext(ctx, query, feedIDsParam)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	plan := &RefreshPlan{}

	for rows.Next() {
		var feedID int
		var currentFvID, materializedFvID sql.NullInt64
		var onestopID, feedName, action string

		err := rows.Scan(&feedID, &currentFvID, &materializedFvID, &onestopID, &feedName, &action)
		if err != nil {
			return nil, err
		}

		change := FeedVersionChange{
			FeedID:    feedID,
			OnestopID: onestopID,
			FeedName:  feedName,
		}

		if materializedFvID.Valid {
			oldID := int(materializedFvID.Int64)
			change.OldFeedVersionID = &oldID
		}
		if currentFvID.Valid {
			change.NewFeedVersionID = int(currentFvID.Int64)
		}

		switch action {
		case "add":
			plan.FeedsToAdd = append(plan.FeedsToAdd, change)
		case "remove":
			plan.FeedsToRemove = append(plan.FeedsToRemove, change)
		case "update":
			plan.FeedsToUpdate = append(plan.FeedsToUpdate, change)
		}
	}

	return plan, nil
}

// clearAllMaterializedData removes all data from materialized tables
func (r *Refresher) clearAllMaterializedData(ctx context.Context, atx tldb.Adapter) error {
	tables := []string{
		"tl_materialized_active_routes",
		"tl_materialized_active_stops",
		"tl_materialized_active_agencies",
		"tl_materialized_index_state",
	}

	for _, table := range tables {
		_, err := atx.Sqrl().Delete(table).ExecContext(ctx)
		if err != nil {
			return fmt.Errorf("failed to clear table %s: %w", table, err)
		}
	}

	return nil
}

// removeFeedVersion removes all routes/stops/agencies for a feed from materialized tables
func (r *Refresher) removeFeedVersion(ctx context.Context, atx tldb.Adapter, change FeedVersionChange) error {
	// Remove routes
	_, err := atx.Sqrl().
		Delete("tl_materialized_active_routes").
		Where(sq.Eq{"feed_id": change.FeedID}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove routes for feed %d: %w", change.FeedID, err)
	}

	// Remove stops
	_, err = atx.Sqrl().
		Delete("tl_materialized_active_stops").
		Where(sq.Eq{"feed_id": change.FeedID}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove stops for feed %d: %w", change.FeedID, err)
	}

	// Remove agencies
	_, err = atx.Sqrl().
		Delete("tl_materialized_active_agencies").
		Where(sq.Eq{"feed_id": change.FeedID}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove agencies for feed %d: %w", change.FeedID, err)
	}

	// Remove state tracking
	_, err = atx.Sqrl().
		Delete("tl_materialized_index_state").
		Where(sq.Eq{"feed_id": change.FeedID}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove state for feed %d: %w", change.FeedID, err)
	}

	return nil
}

// updateFeedVersion replaces routes/stops for a feed version
func (r *Refresher) updateFeedVersion(ctx context.Context, atx tldb.Adapter, change FeedVersionChange) error {
	// Remove old data first
	tempChange := FeedVersionChange{FeedID: change.FeedID}
	if err := r.removeFeedVersion(ctx, atx, tempChange); err != nil {
		return err
	}

	// Add new data
	return r.addFeedVersion(ctx, atx, change)
}

// addFeedVersion inserts routes/stops for a new feed version
func (r *Refresher) addFeedVersion(ctx context.Context, atx tldb.Adapter, change FeedVersionChange) error {
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

	_, err := atx.DBX().ExecContext(ctx, routeInsert, change.NewFeedVersionID)
	if err != nil {
		return fmt.Errorf("failed to insert routes for feed version %d: %w", change.NewFeedVersionID, err)
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

	_, err = atx.DBX().ExecContext(ctx, stopInsert, change.NewFeedVersionID)
	if err != nil {
		return fmt.Errorf("failed to insert stops for feed version %d: %w", change.NewFeedVersionID, err)
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

	_, err = atx.DBX().ExecContext(ctx, agencyInsert, change.NewFeedVersionID)
	if err != nil {
		return fmt.Errorf("failed to insert agencies for feed version %d: %w", change.NewFeedVersionID, err)
	}

	// Update state tracking
	_, err = atx.Sqrl().
		Insert("tl_materialized_index_state").
		Columns("feed_id", "materialized_feed_version_id").
		Values(change.FeedID, change.NewFeedVersionID).
		Suffix("ON CONFLICT (feed_id) DO UPDATE SET materialized_feed_version_id = ?, last_materialized_at = now()", change.NewFeedVersionID).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to update state for feed %d: %w", change.FeedID, err)
	}

	return nil
}

// GetRefreshStats returns statistics about the materialized tables
func (r *Refresher) GetRefreshStats(ctx context.Context) (*RefreshStats, error) {
	stats := &RefreshStats{}

	// Count routes
	err := r.adapter.Get(ctx, &stats.RouteCount, "SELECT count(*) FROM tl_materialized_active_routes")
	if err != nil {
		return nil, err
	}

	// Count stops
	err = r.adapter.Get(ctx, &stats.StopCount, "SELECT count(*) FROM tl_materialized_active_stops")
	if err != nil {
		return nil, err
	}

	// Count agencies
	err = r.adapter.Get(ctx, &stats.AgencyCount, "SELECT count(*) FROM tl_materialized_active_agencies")
	if err != nil {
		return nil, err
	}

	// Count feeds in materialized state
	err = r.adapter.Get(ctx, &stats.FeedCount, "SELECT count(*) FROM tl_materialized_index_state")
	if err != nil {
		return nil, err
	}

	// Count total active feeds
	err = r.adapter.Get(ctx, &stats.TotalFeeds, `
		SELECT count(*) 
		FROM feed_states fs 
		JOIN current_feeds cf ON cf.id = fs.feed_id 
		WHERE fs.feed_version_id IS NOT NULL 
		  AND fs.public = true 
		  AND cf.deleted_at IS NULL
	`)
	if err != nil {
		return nil, err
	}

	// Get last refresh time
	err = r.adapter.Get(ctx, &stats.LastRefresh, "SELECT max(last_materialized_at) FROM tl_materialized_index_state")
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Calculate out of sync count
	stats.OutOfSync = stats.TotalFeeds - stats.FeedCount

	return stats, nil
}
