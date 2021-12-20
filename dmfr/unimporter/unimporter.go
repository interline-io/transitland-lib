package unimporter

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/tldb"
)

func feedVersionTableDelete(atx tldb.Adapter, table string, fvid int) error {
	where := sq.Eq{"feed_version_id": fvid}
	_, err := atx.Sqrl().Delete(table).Where(where).Exec()
	if err != nil {
		return err
	}
	return nil
}

// UnimportSchedule removes schedule data for a feed version and updates the import record.
// stops, routes, agencies, pathways, levels are not affected.
func UnimportSchedule(atx tldb.Adapter, id int) error {
	tables := []string{
		"gtfs_stop_times",
		"gtfs_transfers",
		"gtfs_calendar_dates",
		"gtfs_frequencies",
		"gtfs_trips",
		"gtfs_shapes",
		"gtfs_calendars",
	}
	where := sq.Eq{"feed_version_id": id}
	for _, table := range tables {
		if err := feedVersionTableDelete(atx, table, id); err != nil {
			return err
		}
	}
	if _, err := atx.Sqrl().Update("feed_version_gtfs_imports").Set("schedule_removed", true).Where(where).Exec(); err != nil {
		return err
	}
	return nil
}

// UnimportFeedVersion unimports a feed version and removes the feed_version_gtfs_import record.
func UnimportFeedVersion(atx tldb.Adapter, id int, extraTables []string) error {
	// Set of tables to delete where feed_version_id = fvid
	// Order is important
	tables := []string{
		// derived entities
		"tl_agency_geometries",
		"tl_agency_places",
		"tl_route_geometries",
		"tl_route_stops",
		"tl_route_headways",
		"tl_feed_version_geometries",
		"tl_agency_onestop_ids",
		"tl_route_onestop_ids",
		"tl_stop_onestop_ids",
		// stop times
		"gtfs_stop_times",
		// anonymous entities
		"gtfs_transfers",
		"gtfs_calendar_dates",
		"gtfs_feed_infos",
		"gtfs_frequencies",
		"gtfs_fare_rules",
		// named entities
		"gtfs_pathways",
		"gtfs_fare_attributes",
		"gtfs_trips",
		"gtfs_shapes",
		"gtfs_calendars",
		"gtfs_routes",
		"gtfs_stops",
		"gtfs_agencies",
		"gtfs_levels",
	}
	// Run in txn
	where := sq.Eq{"feed_version_id": id}
	for _, table := range extraTables {
		_, err := atx.Sqrl().Delete(table).Where(where).Exec()
		if err != nil {
			return err
		}
	}
	for _, table := range tables {
		if err := feedVersionTableDelete(atx, table, id); err != nil {
			return err
		}
	}
	if _, err := atx.Sqrl().Delete("feed_version_gtfs_imports").Where(where).Exec(); err != nil {
		return err
	}
	if _, err := atx.Sqrl().Update("feed_states").Set("feed_version_id", nil).Where(where).Exec(); err != nil {
		return err
	}
	return nil
}
