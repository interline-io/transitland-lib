package feedstate

import (
	"context"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/internal/testreader"
	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
	_ "github.com/interline-io/transitland-lib/tldb/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testFeedOnestopID  = "f-test~feed"
	testFeedOnestopID2 = "f-test~feed2"
	testFeedName2      = "Test Feed 2"
)

// setupTestDB creates an in-memory SQLite database with test GTFS data
// If url is provided, creates a feed_version and imports the GTFS data
// Returns (adapter, feedID, feedVersionID) where feedVersionID is 0 if no URL provided
func setupTestDB(t *testing.T, onestopID string, url string) (tldb.Adapter, int, int) {
	t.Helper() // Mark this as a helper function for better test reporting

	// Create in-memory SQLite database
	writer, err := tldb.OpenWriter("sqlite3://:memory:", true)
	require.NoError(t, err, "failed to open database writer")

	adapter := writer.Adapter

	// Create a feed with deterministic ID
	const feedID = 1
	feedName := "Test Feed"

	ctx := context.Background()

	// Use Squirrel to insert feed
	_, err = adapter.Sqrl().
		Insert("current_feeds").
		Columns("id", "onestop_id", "name").
		Values(feedID, onestopID, feedName).
		RunWith(adapter.DBX()).
		ExecContext(ctx)
	require.NoError(t, err, "failed to create test feed")

	// Use Squirrel to insert feed_state
	_, err = adapter.Sqrl().
		Insert("feed_states").
		Columns("feed_id", "feed_version_id", "public", "feed_realtime_enabled").
		Values(feedID, nil, 1, 0).
		RunWith(adapter.DBX()).
		ExecContext(ctx)
	require.NoError(t, err, "failed to create feed state")

	// If no URL provided, return without creating feed version
	if url == "" {
		return adapter, feedID, 0
	}

	// Create a feed version using Squirrel
	var feedVersionID int
	query, args, err := adapter.Sqrl().
		Insert("feed_versions").
		Columns("feed_id", "sha1", "name").
		Values(feedID, "test-sha1", "Test Feed Version").
		Suffix("RETURNING id").
		ToSql()
	require.NoError(t, err, "failed to build feed version insert query")

	err = adapter.DBX().QueryRowxContext(ctx, query, args...).Scan(&feedVersionID)
	require.NoError(t, err, "failed to create feed version")

	// Import GTFS data from provided URL
	reader, err := tlcsv.NewReader(url)
	require.NoError(t, err, "failed to create GTFS reader")

	writer.FeedVersionID = feedVersionID
	result, err := copier.CopyWithOptions(ctx, reader, writer, copier.Options{})
	require.NoError(t, err, "failed to import GTFS data")
	require.Empty(t, result.Errors, "import should have no errors")

	return adapter, feedID, feedVersionID
}

// Helper function to verify materialized table counts
func verifyMaterializedCounts(t *testing.T, adapter tldb.Adapter, feedVersionID int, expectedCounts map[string]int) {
	t.Helper()
	ctx := context.Background()

	tables := map[string]string{
		"routes":   "tl_materialized_active_routes",
		"stops":    "tl_materialized_active_stops",
		"agencies": "tl_materialized_active_agencies",
	}

	for entity, table := range tables {
		var count int
		err := dbutil.Get(ctx, adapter.DBX(), adapter.Sqrl().
			Select("COUNT(*)").
			From(table).
			Where("feed_version_id = ?", feedVersionID), &count)
		require.NoError(t, err, "failed to count materialized %s", entity)

		if expected, exists := expectedCounts[entity]; exists {
			assert.Equal(t, expected, count, "wrong %s count", entity)
		} else {
			assert.Greater(t, count, 0, "should have materialized %s", entity)
		}
	}
}

// Helper function to verify feed state
func verifyFeedState(t *testing.T, adapter tldb.Adapter, feedID int, expectedFeedVersionID *int) {
	t.Helper()
	ctx := context.Background()

	var activeFeedVersionID *int
	err := dbutil.Get(ctx, adapter.DBX(), adapter.Sqrl().
		Select("feed_version_id").
		From("feed_states").
		Where("feed_id = ?", feedID), &activeFeedVersionID)
	require.NoError(t, err, "failed to query feed_states")

	if expectedFeedVersionID == nil {
		assert.Nil(t, activeFeedVersionID, "feed_version_id should be null")
	} else {
		require.NotNil(t, activeFeedVersionID, "feed_version_id should not be null")
		assert.Equal(t, *expectedFeedVersionID, *activeFeedVersionID, "wrong feed version ID in feed_states")
	}
}

func TestManager_ActivateFeedVersion(t *testing.T) {
	adapter, feedID, feedVersionID := setupTestDB(t, testFeedOnestopID, testreader.ExampleZip.URL)
	defer adapter.Close()

	manager := NewManager(adapter)
	ctx := context.Background()

	// Test activating a feed version
	err := manager.ActivateFeedVersion(ctx, feedVersionID)
	require.NoError(t, err, "failed to activate feed version")

	// Verify feed_states was updated
	verifyFeedState(t, adapter, feedID, &feedVersionID)

	// Verify materialized tables were populated
	verifyMaterializedCounts(t, adapter, feedVersionID, map[string]int{})

	// Test activating the same version again (should be no-op)
	err = manager.ActivateFeedVersion(ctx, feedVersionID)
	require.NoError(t, err, "re-activating same version should succeed")
}

func TestManager_DeactivateFeedVersion(t *testing.T) {
	adapter, feedID, feedVersionID := setupTestDB(t, testFeedOnestopID, testreader.ExampleZip.URL)
	defer adapter.Close()

	manager := NewManager(adapter)
	ctx := context.Background()

	// First activate the feed version
	err := manager.ActivateFeedVersion(ctx, feedVersionID)
	require.NoError(t, err, "failed to activate feed version")

	// Verify it's active
	verifyFeedState(t, adapter, feedID, &feedVersionID)

	// Now deactivate it
	err = manager.DeactivateFeedVersion(ctx, feedVersionID)
	require.NoError(t, err, "failed to deactivate feed version")

	// Verify feed_states was updated (should be null)
	verifyFeedState(t, adapter, feedID, nil)

	// Verify materialized tables were cleared
	verifyMaterializedCounts(t, adapter, feedVersionID, map[string]int{
		"routes":   0,
		"stops":    0,
		"agencies": 0,
	})

	// Test deactivating an inactive version (should be no-op)
	err = manager.DeactivateFeedVersion(ctx, feedVersionID)
	require.NoError(t, err, "deactivating inactive version should succeed")
}

func TestManager_GetActiveFeedVersions(t *testing.T) {
	adapter, _, feedVersionID := setupTestDB(t, testFeedOnestopID, testreader.ExampleZip.URL)
	defer adapter.Close()

	manager := NewManager(adapter)
	ctx := context.Background()

	// Initially no active feed versions
	active, err := manager.GetActiveFeedVersions(ctx)
	require.NoError(t, err, "failed to get active feed versions")
	assert.Empty(t, active, "should have no active feed versions initially")

	// Activate a feed version
	err = manager.ActivateFeedVersion(ctx, feedVersionID)
	require.NoError(t, err, "failed to activate feed version")

	// Should now see the active feed version
	active, err = manager.GetActiveFeedVersions(ctx)
	require.NoError(t, err, "failed to get active feed versions")
	assert.Len(t, active, 1, "should have one active feed version")
	assert.Equal(t, feedVersionID, active[0], "wrong feed version ID")
}

func TestManager_SetActiveFeedVersions(t *testing.T) {
	adapter, _, feedVersionID := setupTestDB(t, testFeedOnestopID, testreader.ExampleZip.URL)
	defer adapter.Close()

	manager := NewManager(adapter)
	ctx := context.Background()

	// Create a second feed and feed version
	feedID2 := 2
	_, err := adapter.Sqrl().
		Insert("current_feeds").
		Columns("id", "onestop_id", "name").
		Values(feedID2, testFeedOnestopID2, testFeedName2).
		RunWith(adapter.DBX()).
		ExecContext(ctx)
	require.NoError(t, err, "failed to create second test feed")

	_, err = adapter.Sqrl().
		Insert("feed_states").
		Columns("feed_id", "feed_version_id", "public", "feed_realtime_enabled").
		Values(feedID2, nil, 1, 0).
		RunWith(adapter.DBX()).
		ExecContext(ctx)
	require.NoError(t, err, "failed to create second feed state")

	// Create a second feed version for the second feed
	_, err = adapter.Sqrl().
		Insert("feed_versions").
		Columns("id", "feed_id", "sha1", "fetched_at", "url", "earliest_calendar_date", "latest_calendar_date").
		Values(feedVersionID+1, feedID2, "test-sha2", time.Now(), "test-url-2", "2024-01-01", "2024-12-31").
		RunWith(adapter.DBX()).
		ExecContext(ctx)
	require.NoError(t, err, "failed to create second feed version")

	feedVersionID2 := feedVersionID + 1

	// Import data for second feed (copy the same GTFS data with different feed_id)
	reader, err := tlcsv.NewReader(testreader.ExampleFeedCaltrain.URL)
	require.NoError(t, err, "failed to create GTFS reader for second feed")

	writer, err := tldb.NewWriter("sqlite3://:memory:")
	require.NoError(t, err, "failed to create database writer for second feed")
	writer.Adapter = adapter
	writer.FeedVersionID = feedVersionID2

	result, err := copier.CopyWithOptions(ctx, reader, writer, copier.Options{})
	require.NoError(t, err, "failed to import GTFS data for second feed")
	require.Empty(t, result.Errors, "import should have no errors")

	// Test setting active feed versions
	targetVersions := []int{feedVersionID, feedVersionID2}
	err = manager.SetActiveFeedVersions(ctx, targetVersions)
	require.NoError(t, err, "failed to set active feed versions")

	// Verify both are now active
	active, err := manager.GetActiveFeedVersions(ctx)
	require.NoError(t, err, "failed to get active feed versions")
	assert.Len(t, active, 2, "should have two active feed versions")
	assert.ElementsMatch(t, targetVersions, active, "wrong active feed versions")

	// Test setting to just one feed version (should deactivate the other)
	err = manager.SetActiveFeedVersions(ctx, []int{feedVersionID})
	require.NoError(t, err, "failed to set active feed versions to single version")

	active, err = manager.GetActiveFeedVersions(ctx)
	require.NoError(t, err, "failed to get active feed versions")
	assert.Len(t, active, 1, "should have one active feed version")
	assert.Equal(t, feedVersionID, active[0], "wrong active feed version")

	// Verify the second feed version was deactivated
	var count int
	err = dbutil.Get(ctx, adapter.DBX(), adapter.Sqrl().
		Select("COUNT(*)").
		From("tl_materialized_active_routes").
		Where("feed_version_id = ?", feedVersionID2), &count)
	require.NoError(t, err, "failed to count materialized routes for second feed")
	assert.Equal(t, 0, count, "second feed should be dematerialized")

	// Test setting to empty list (should deactivate all)
	err = manager.SetActiveFeedVersions(ctx, []int{})
	require.NoError(t, err, "failed to deactivate all feed versions")

	active, err = manager.GetActiveFeedVersions(ctx)
	require.NoError(t, err, "failed to get active feed versions")
	assert.Empty(t, active, "should have no active feed versions")
}

func TestManager_MaterializedDataIntegrity(t *testing.T) {
	adapter, _, feedVersionID := setupTestDB(t, testFeedOnestopID, testreader.ExampleZip.URL)
	defer adapter.Close()

	manager := NewManager(adapter)
	ctx := context.Background()

	// Activate the feed version
	err := manager.ActivateFeedVersion(ctx, feedVersionID)
	require.NoError(t, err, "failed to activate feed version")

	// Check that materialized data matches source data
	testCases := []struct {
		entity            string
		sourceTable       string
		materializedTable string
	}{
		{"routes", "gtfs_routes", "tl_materialized_active_routes"},
		{"stops", "gtfs_stops", "tl_materialized_active_stops"},
		{"agencies", "gtfs_agencies", "tl_materialized_active_agencies"},
	}

	for _, tc := range testCases {
		t.Run(tc.entity, func(t *testing.T) {
			var sourceCount, materializedCount int

			// Query source table count using dbutil.Get
			err := dbutil.Get(ctx, adapter.DBX(), adapter.Sqrl().
				Select("COUNT(*)").
				From(tc.sourceTable).
				Where("feed_version_id = ?", feedVersionID), &sourceCount)
			require.NoError(t, err, "failed to count source %s", tc.entity)

			// Query materialized table count using dbutil.Get
			err = dbutil.Get(ctx, adapter.DBX(), adapter.Sqrl().
				Select("COUNT(*)").
				From(tc.materializedTable).
				Where("feed_version_id = ?", feedVersionID), &materializedCount)
			require.NoError(t, err, "failed to count materialized %s", tc.entity)

			assert.Equal(t, sourceCount, materializedCount,
				"materialized %s count should match source", tc.entity)
		})
	}

	// Test specific data integrity - check that route data was copied correctly
	type routeData struct {
		RouteID        string `db:"route_id"`
		RouteShortName string `db:"route_short_name"`
		RouteLongName  string `db:"route_long_name"`
		RouteType      int    `db:"route_type"`
		AgencyName     string `db:"agency_name"`
	}

	var sourceRoutes []routeData
	err = adapter.Select(ctx, &sourceRoutes, `
		SELECT gtfs_routes.route_id, gtfs_routes.route_short_name, gtfs_routes.route_long_name, 
		       gtfs_routes.route_type, gtfs_agencies.agency_name
		FROM gtfs_routes 
		JOIN gtfs_agencies ON gtfs_agencies.id = gtfs_routes.agency_id
		WHERE gtfs_routes.feed_version_id = ?
		ORDER BY gtfs_routes.route_id`, feedVersionID)
	require.NoError(t, err, "failed to query source routes")

	var materializedRoutes []routeData
	err = adapter.Select(ctx, &materializedRoutes, `
		SELECT route_id, route_short_name, route_long_name, route_type, agency_name
		FROM tl_materialized_active_routes 
		WHERE feed_version_id = ?
		ORDER BY route_id`, feedVersionID)
	require.NoError(t, err, "failed to query materialized routes")

	assert.Equal(t, sourceRoutes, materializedRoutes, "materialized route data should match source data")
}

func TestManager_GetFeedIDForFeedVersion(t *testing.T) {
	adapter, feedID, feedVersionID := setupTestDB(t, testFeedOnestopID, testreader.ExampleZip.URL)
	defer adapter.Close()

	manager := NewManager(adapter)
	ctx := context.Background()

	// Test getting feed ID for feed version
	resultFeedID, err := manager.GetFeedIDForFeedVersion(ctx, feedVersionID)
	require.NoError(t, err, "failed to get feed ID")
	assert.Equal(t, feedID, resultFeedID, "wrong feed ID returned")

	// Test with non-existent feed version
	_, err = manager.GetFeedIDForFeedVersion(ctx, 99999)
	assert.Error(t, err, "should error for non-existent feed version")
}
