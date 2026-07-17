package feedstate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
	sq "github.com/irees/squirrel"
)

// Manager handles feed state and materialized table operations
//
// ActivateFeedVersion and DeactivateFeedVersion are self-transactional, joining the caller's
// transaction if one is open, and SetActiveFeedVersions transacts per feed version through them.
// Other methods leave transactions to the caller.
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
//
// Runs in a transaction: the feed_states swap and materialized table rebuild must not be
// observed half done.
func (m *Manager) ActivateFeedVersion(ctx context.Context, feedVersionID int) error {
	return m.adapter.Tx(func(atx tldb.Adapter) error {
		return (&Manager{adapter: atx}).activateFeedVersion(ctx, feedVersionID)
	})
}

func (m *Manager) activateFeedVersion(ctx context.Context, feedVersionID int) error {
	feedID, err := m.GetFeedIDForFeedVersion(ctx, feedVersionID)
	if err != nil {
		return fmt.Errorf("failed to get feed_id for feed version %d: %w", feedVersionID, err)
	}
	log.For(ctx).Info().
		Int("feed_version_id", feedVersionID).
		Int("feed_id", feedID).
		Msg("Activating feed version")
	return m.reconcileFeed(ctx, feedID, tt.NewInt(feedVersionID))
}

// feedStatePointers is the pointer/visibility state the reconciler derives from.
type feedStatePointers struct {
	ActiveFeedVersionID       tt.Int `db:"active_feed_version_id"`
	MaterializedFeedVersionID tt.Int `db:"materialized_feed_version_id"`
	ExcludeFromGlobal         bool   `db:"exclude_from_global"`
}

func (m *Manager) getFeedStatePointers(ctx context.Context, feedID int) (feedStatePointers, error) {
	var fs feedStatePointers
	err := dbutil.Get(ctx, m.adapter.DBX(), m.adapter.Sqrl().
		Select("active_feed_version_id", "materialized_feed_version_id", "exclude_from_global").
		From("feed_states").
		Where(sq.Eq{"feed_id": feedID}), &fs)
	if errors.Is(err, sql.ErrNoRows) {
		// No feed_states row yet: no active/materialized version, not excluded.
		return feedStatePointers{}, nil
	}
	return fs, err
}

// reconcileFeed sets a feed's active feed version and brings its materialized pointer and
// materialized tables into line. materialized_feed_version_id is the active version unless the
// feed is excluded from global queries, in which case it is null; feed_version_id is written as a
// transitional mirror of it. Materialized rows change only when the visible version does. Assumes
// an open transaction.
func (m *Manager) reconcileFeed(ctx context.Context, feedID int, active tt.Int) error {
	cur, err := m.getFeedStatePointers(ctx, feedID)
	if err != nil {
		return fmt.Errorf("failed to read feed_states for feed %d: %w", feedID, err)
	}

	materialized := active
	if cur.ExcludeFromGlobal {
		materialized = tt.Int{}
	}

	// Reconcile materialized tables only when the visible version actually changes.
	old := cur.MaterializedFeedVersionID
	if old != materialized {
		if old.Valid {
			if err := m.DematerializeFeedVersion(ctx, old.Int()); err != nil {
				return err
			}
		}
		if materialized.Valid {
			if err := m.MaterializeFeedVersion(ctx, materialized.Int()); err != nil {
				return err
			}
		}
	}

	_, err = m.adapter.Sqrl().
		Update("feed_states").
		Set("active_feed_version_id", active).
		Set("materialized_feed_version_id", materialized).
		Set("feed_version_id", materialized). // TRANSITIONAL: mirror of materialized_feed_version_id; delete with the column
		Where(sq.Eq{"feed_id": feedID}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to update feed_states for feed %d: %w", feedID, err)
	}
	return nil
}

// DeactivateFeedVersion deactivates a feed version by removing it from feed_states and materialized tables
// If the feed version is not currently active, does nothing
func (m *Manager) DeactivateFeedVersion(ctx context.Context, feedVersionID int) error {
	return m.adapter.Tx(func(atx tldb.Adapter) error {
		return (&Manager{adapter: atx}).deactivateFeedVersion(ctx, feedVersionID)
	})
}

func (m *Manager) deactivateFeedVersion(ctx context.Context, feedVersionID int) error {
	feedID, err := m.GetFeedIDForFeedVersion(ctx, feedVersionID)
	if err != nil {
		return fmt.Errorf("failed to get feed_id for feed version %d: %w", feedVersionID, err)
	}
	cur, err := m.getFeedStatePointers(ctx, feedID)
	if err != nil {
		return fmt.Errorf("failed to read feed_states for feed %d: %w", feedID, err)
	}
	// Only clear when this feed version is the feed's current active version.
	if !cur.ActiveFeedVersionID.Valid || cur.ActiveFeedVersionID.Int() != feedVersionID {
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
	return m.reconcileFeed(ctx, feedID, tt.Int{})
}

// SetExcludeFromGlobal sets a feed's global-query visibility and reconciles its materialized
// pointer and materialized tables: excluding dematerializes the active version, including
// materializes it. Runs in a transaction.
func (m *Manager) SetExcludeFromGlobal(ctx context.Context, feedID int, exclude bool) error {
	return m.adapter.Tx(func(atx tldb.Adapter) error {
		mm := &Manager{adapter: atx}
		if _, err := mm.adapter.Sqrl().
			Update("feed_states").
			Set("exclude_from_global", exclude).
			Where(sq.Eq{"feed_id": feedID}).
			ExecContext(ctx); err != nil {
			return fmt.Errorf("failed to set exclude_from_global for feed %d: %w", feedID, err)
		}
		cur, err := mm.getFeedStatePointers(ctx, feedID)
		if err != nil {
			return err
		}
		return mm.reconcileFeed(ctx, feedID, cur.ActiveFeedVersionID)
	})
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

// sortedColumnsAndSelects converts a map of column->expression into sorted parallel slices
func sortedColumnsAndSelects(fields map[string]string) ([]string, []string) {
	columns := slices.Sorted(maps.Keys(fields))
	selects := make([]string, len(columns))
	for i, col := range columns {
		selects[i] = fields[col]
	}
	return columns, selects
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

// routeMaterializeFields returns the destination-column -> source-expression
// projection used to populate tl_materialized_active_routes. When spatial is
// true the geometry is simplified (PostGIS); otherwise it is copied as-is (SQLite).
func routeMaterializeFields(spatial bool) map[string]string {
	fields := map[string]string{
		"id":                  "gtfs_routes.id",
		"route_id":            "gtfs_routes.route_id",
		"route_short_name":    "gtfs_routes.route_short_name",
		"route_long_name":     "gtfs_routes.route_long_name",
		"route_desc":          "gtfs_routes.route_desc",
		"route_type":          "gtfs_routes.route_type",
		"route_url":           "gtfs_routes.route_url",
		"route_color":         "gtfs_routes.route_color",
		"route_text_color":    "gtfs_routes.route_text_color",
		"route_sort_order":    "gtfs_routes.route_sort_order",
		"network_id":          "gtfs_routes.network_id",
		"as_route":            "gtfs_routes.as_route",
		"continuous_pickup":   "gtfs_routes.continuous_pickup",
		"continuous_drop_off": "gtfs_routes.continuous_drop_off",
		"cemv_support":        "gtfs_routes.cemv_support",
		"agency_id":           "gtfs_routes.agency_id",
		"gtfs_agency_id":      "gtfs_agencies.agency_id",
		"agency_name":         "gtfs_agencies.agency_name",
		"feed_version_id":     "feed_versions.id",
		"feed_id":             "feed_versions.feed_id",
		"onestop_id":          "osid.onestop_id",
		"textsearch":          "gtfs_routes.textsearch",
		"created_at":          "gtfs_routes.created_at",
		"updated_at":          "gtfs_routes.updated_at",
	}
	// Geometry column - full geometry for SQLite, simplified for PostGIS
	fields["geometry_simplified"] = "tlrg.geometry"
	if spatial {
		fields["geometry_simplified"] = "ST_Simplify(tlrg.geometry::geometry, 0.01)"
	}
	return fields
}

// stopMaterializeFields returns the destination-column -> source-expression
// projection used to populate tl_materialized_active_stops.
func stopMaterializeFields() map[string]string {
	return map[string]string{
		"id":                  "gtfs_stops.id",
		"stop_id":             "gtfs_stops.stop_id",
		"stop_code":           "gtfs_stops.stop_code",
		"stop_name":           "gtfs_stops.stop_name",
		"stop_desc":           "gtfs_stops.stop_desc",
		"zone_id":             "gtfs_stops.zone_id",
		"stop_url":            "gtfs_stops.stop_url",
		"location_type":       "gtfs_stops.location_type",
		"stop_timezone":       "gtfs_stops.stop_timezone",
		"wheelchair_boarding": "gtfs_stops.wheelchair_boarding",
		"parent_station":      "gtfs_stops.parent_station",
		"level_id":            "gtfs_stops.level_id",
		"area_id":             "gtfs_stops.area_id",
		"tts_stop_name":       "gtfs_stops.tts_stop_name",
		"platform_code":       "gtfs_stops.platform_code",
		"stop_access":         "gtfs_stops.stop_access",
		"feed_version_id":     "feed_versions.id",
		"feed_id":             "feed_versions.feed_id",
		"onestop_id":          "osid.onestop_id",
		"geometry":            "gtfs_stops.geometry",
		"textsearch":          "gtfs_stops.textsearch",
		"created_at":          "gtfs_stops.created_at",
		"updated_at":          "gtfs_stops.updated_at",
	}
}

// agencyMaterializeFields returns the destination-column -> source-expression
// projection used to populate tl_materialized_active_agencies.
func agencyMaterializeFields() map[string]string {
	return map[string]string{
		"id":              "gtfs_agencies.id",
		"agency_id":       "gtfs_agencies.agency_id",
		"agency_name":     "gtfs_agencies.agency_name",
		"agency_url":      "gtfs_agencies.agency_url",
		"agency_timezone": "gtfs_agencies.agency_timezone",
		"agency_lang":     "gtfs_agencies.agency_lang",
		"agency_phone":    "gtfs_agencies.agency_phone",
		"agency_fare_url": "gtfs_agencies.agency_fare_url",
		"agency_email":    "gtfs_agencies.agency_email",
		"cemv_support":    "gtfs_agencies.cemv_support",
		"feed_version_id": "feed_versions.id",
		"feed_id":         "feed_versions.feed_id",
		"onestop_id":      "osid.onestop_id",
		"textsearch":      "gtfs_agencies.textsearch",
		"created_at":      "gtfs_agencies.created_at",
		"updated_at":      "gtfs_agencies.updated_at",
	}
}

// MaterializedTableFields returns, for each materialized active table, the
// destination-column -> source-expression projection used by
// MaterializeFeedVersion. These maps are the single source of truth for what
// those tables must contain; schema-drift checks assert that every destination
// column exists in the corresponding table.
func MaterializedTableFields(spatial bool) map[string]map[string]string {
	return map[string]map[string]string{
		"tl_materialized_active_routes":   routeMaterializeFields(spatial),
		"tl_materialized_active_stops":    stopMaterializeFields(),
		"tl_materialized_active_agencies": agencyMaterializeFields(),
	}
}

// MaterializeFeedVersion inserts routes/stops/agencies for a feed version into materialized tables
func (m *Manager) MaterializeFeedVersion(ctx context.Context, feedVersionID int) error {
	// Clear any existing materialized data for this feed version first
	if err := m.DematerializeFeedVersion(ctx, feedVersionID); err != nil {
		return fmt.Errorf("failed to dematerialize feed version %d before materializing: %w", feedVersionID, err)
	}

	// Build route column mappings (destination column -> source expression)
	routeFields := routeMaterializeFields(m.adapter.SupportsSpatialFunctions())

	// Extract columns and selects from the map, sorted for consistency
	routeColumns, routeSelects := sortedColumnsAndSelects(routeFields)

	// Insert routes with derived data using Squirrel
	routeQuery := m.adapter.Sqrl().
		Insert("tl_materialized_active_routes").
		Columns(routeColumns...).
		Select(m.adapter.Sqrl().
			Select(routeSelects...).
			From("gtfs_routes").
			Join("gtfs_agencies ON gtfs_agencies.id = gtfs_routes.agency_id").
			Join("feed_versions ON feed_versions.id = gtfs_routes.feed_version_id").
			LeftJoin("feed_version_route_onestop_ids osid ON osid.entity_id = gtfs_routes.route_id AND osid.feed_version_id = feed_versions.id").
			LeftJoin("tl_route_geometries tlrg ON tlrg.route_id = gtfs_routes.id").
			Where(sq.Eq{"gtfs_routes.feed_version_id": feedVersionID}))

	_, err := routeQuery.ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert routes for feed version %d: %w", feedVersionID, err)
	}

	// Build stop column mappings (destination column -> source expression)
	stopFields := stopMaterializeFields()

	// Extract columns and selects from the map, sorted for consistency
	stopColumns, stopSelects := sortedColumnsAndSelects(stopFields)

	// Insert stops with derived data using Squirrel
	stopQuery := m.adapter.Sqrl().
		Insert("tl_materialized_active_stops").
		Columns(stopColumns...).
		Select(m.adapter.Sqrl().
			Select(stopSelects...).
			From("gtfs_stops").
			Join("feed_versions ON feed_versions.id = gtfs_stops.feed_version_id").
			LeftJoin("feed_version_stop_onestop_ids osid ON osid.entity_id = gtfs_stops.stop_id AND osid.feed_version_id = feed_versions.id").
			Where(sq.Eq{"gtfs_stops.feed_version_id": feedVersionID}))

	_, err = stopQuery.ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert stops for feed version %d: %w", feedVersionID, err)
	}

	// Build agency column mappings (destination column -> source expression)
	agencyFields := agencyMaterializeFields()

	// Extract columns and selects from the map, sorted for consistency
	agencyColumns, agencySelects := sortedColumnsAndSelects(agencyFields)

	// Insert agencies with derived data using Squirrel
	agencyQuery := m.adapter.Sqrl().
		Insert("tl_materialized_active_agencies").
		Columns(agencyColumns...).
		Select(m.adapter.Sqrl().
			Select(agencySelects...).
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

// GetActiveFeedVersions returns the active_feed_version_id of every feed that has one.
func (m *Manager) GetActiveFeedVersions(ctx context.Context) ([]int, error) {
	var ids []int
	err := dbutil.Select(ctx, m.adapter.DBX(), m.adapter.Sqrl().
		Select("active_feed_version_id").
		From("feed_states").
		Where("active_feed_version_id IS NOT NULL"), &ids)
	if err != nil {
		return nil, fmt.Errorf("failed to query active feed versions: %w", err)
	}
	sort.Ints(ids)
	return ids, nil
}

// GetMaterializedFeedVersionPointers returns the materialized_feed_version_id of every feed that
// has one — the set that should be present in the materialized tables.
func (m *Manager) GetMaterializedFeedVersionPointers(ctx context.Context) ([]int, error) {
	var ids []int
	err := dbutil.Select(ctx, m.adapter.DBX(), m.adapter.Sqrl().
		Select("materialized_feed_version_id").
		From("feed_states").
		Where("materialized_feed_version_id IS NOT NULL"), &ids)
	if err != nil {
		return nil, fmt.Errorf("failed to query materialized feed versions: %w", err)
	}
	sort.Ints(ids)
	return ids, nil
}

type FeedVersionChanges struct {
	ToDeactivate []int
	ToActivate   []int
}

func (m *Manager) CalculateSetActiveChanges(ctx context.Context, feedVersionIDs []int) (FeedVersionChanges, error) {
	ret := FeedVersionChanges{}
	// Get currently active feed versions
	currentActive, err := m.GetActiveFeedVersions(ctx)
	if err != nil {
		return ret, fmt.Errorf("failed to get current active feed versions: %w", err)
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

	var toDeactivate []int
	for _, fvid := range currentActive {
		if !targetSet[fvid] {
			toDeactivate = append(toDeactivate, fvid)
		}
	}
	var toActivate []int
	for _, fvid := range feedVersionIDs {
		if !currentSet[fvid] {
			toActivate = append(toActivate, fvid)
		}
	}
	return FeedVersionChanges{ToDeactivate: toDeactivate, ToActivate: toActivate}, nil
}

// SetActiveFeedVersions activates the specified feed versions and deactivates all others
// This replaces the entire set of active feed versions with the provided list
func (m *Manager) SetActiveFeedVersions(ctx context.Context, feedVersionIDs []int) error {
	changes, err := m.CalculateSetActiveChanges(ctx, feedVersionIDs)
	if err != nil {
		return fmt.Errorf("failed to calculate changes: %w", err)
	}

	// Deactivate feed versions that should no longer be active
	for _, fvid := range changes.ToDeactivate {
		log.For(ctx).Info().
			Int("feed_version_id", fvid).
			Msg("Deactivating feed version")
		if err := m.DeactivateFeedVersion(ctx, fvid); err != nil {
			return fmt.Errorf("failed to deactivate feed version %d: %w", fvid, err)
		}
	}

	// Activate new feed versions
	for _, fvid := range changes.ToActivate {
		log.For(ctx).Info().
			Int("feed_version_id", fvid).
			Msg("Activating feed version")
		if err := m.ActivateFeedVersion(ctx, fvid); err != nil {
			return fmt.Errorf("failed to activate feed version %d: %w", fvid, err)
		}
	}

	return nil
}
