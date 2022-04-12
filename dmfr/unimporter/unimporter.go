package unimporter

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/tldb"
)

func feedVersionTableDelete(atx tldb.Adapter, table string, fvid int, ifExists bool) error {
	// check if table exists before proceeding
	if ifExists {
		ok, err := atx.TableExists(table)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}
	where := sq.Eq{"feed_version_id": fvid}
	_, err := atx.Sqrl().Delete(table).Where(where).Exec()
	if err != nil {
		return err
	}
	return nil
}

// UnimportSchedule removes schedule data for a feed version and updates the import record.
// stops, routes, agencies, pathways, levels are not affected.
// Note: calendars and calendar_dates MAY be deleted in future versions.
func UnimportSchedule(atx tldb.Adapter, id int) error {
	extensionTables := []string{
		// can reference entities below
		"ext_plus_calendar_attributes",
		"ext_plus_realtime_stops",
		"ext_plus_realtime_trips",
		"ext_plus_timepoints",
	}
	tables := []string{
		// gtfs entities
		"gtfs_attributions",
		"gtfs_stop_times",
		"gtfs_transfers",
		"gtfs_calendar_dates",
		"gtfs_frequencies",
		"gtfs_trips",
		"gtfs_shapes",
		"gtfs_calendars",
	}
	// Allow extension tables to not exist
	for _, table := range extensionTables {
		if err := feedVersionTableDelete(atx, table, id, true); err != nil {
			return err
		}
	}
	for _, table := range tables {
		if err := feedVersionTableDelete(atx, table, id, false); err != nil {
			return err
		}
	}
	where := sq.Eq{"feed_version_id": id}
	if _, err := atx.Sqrl().Update("feed_version_gtfs_imports").Set("schedule_removed", true).Where(where).Exec(); err != nil {
		return err
	}
	return nil
}

// UnimportFeedVersion unimports a feed version and removes the feed_version_gtfs_import record.
func UnimportFeedVersion(atx tldb.Adapter, id int, extraTables []string) error {
	// Set of tables to delete where feed_version_id = fvid
	// Table order is very important
	// built in extensions
	extensionTables := []string{
		"ext_faresv2_areas",
		"ext_faresv2_fare_capping",
		"ext_faresv2_fare_containers",
		"ext_faresv2_fare_leg_rules",
		"ext_faresv2_fare_products",
		"ext_faresv2_fare_timeframes",
		"ext_faresv2_fare_transfer_rules",
		"ext_faresv2_rider_categories",
		"ext_plus_calendar_attributes",
		"ext_plus_directions",
		"ext_plus_fare_rider_categories",
		"ext_plus_farezone_attributes",
		"ext_plus_realtime_routes",
		"ext_plus_realtime_stops",
		"ext_plus_realtime_trips",
		"ext_plus_rider_categories",
		"ext_plus_stop_attributes",
		"ext_plus_timepoints",
	}
	// derived entities
	derivedTables := []string{
		"tl_agency_geometries",
		"tl_agency_places",
		"tl_route_geometries",
		"tl_route_stops",
		"tl_route_headways",
		"tl_feed_version_geometries",
		"tl_agency_onestop_ids",
		"tl_route_onestop_ids",
		"tl_stop_onestop_ids",
	}
	// anonymous entities
	anonTables := []string{
		"tl_stop_external_references",
		"gtfs_stop_times",
		"gtfs_transfers",
		"gtfs_calendar_dates",
		"gtfs_feed_infos",
		"gtfs_frequencies",
		"gtfs_fare_rules",
		"gtfs_attributions",
		"gtfs_translations",
	}
	// named entities
	namedTables := []string{
		"gtfs_pathways",
		"gtfs_fare_attributes",
		"gtfs_trips",
		"gtfs_shapes",
		"gtfs_calendars",
		"gtfs_stops",
		"gtfs_levels",
		"gtfs_routes",
		"gtfs_agencies",
	}
	// Allow extension tables to not exist
	for _, table := range extensionTables {
		if err := feedVersionTableDelete(atx, table, id, true); err != nil {
			return err
		}
	}
	// Other tables must exist
	dt := []string{}
	dt = append(dt, extraTables...)
	dt = append(dt, derivedTables...)
	dt = append(dt, anonTables...)
	dt = append(dt, namedTables...)
	for _, table := range dt {
		if err := feedVersionTableDelete(atx, table, id, false); err != nil {
			return err
		}
	}
	// Remove and cleanup fvgi
	where := sq.Eq{"feed_version_id": id}
	if _, err := atx.Sqrl().Delete("feed_version_gtfs_imports").Where(where).Exec(); err != nil {
		return err
	}
	if _, err := atx.Sqrl().Update("feed_states").Set("feed_version_id", nil).Where(where).Exec(); err != nil {
		return err
	}
	return nil
}
