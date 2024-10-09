package unimporter

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/tldb"
)

// UnimportSchedule removes schedule data for a feed version and updates the import record.
// stops, routes, agencies, pathways, levels are not affected.
// Note: calendars and calendar_dates MAY be deleted in future versions.
func UnimportSchedule(atx tldb.Adapter, id int) error {
	fvt := dmfr.GetFeedVersionTables()
	tables := fvt.ScheduleTables()
	for _, table := range tables {
		if err := dmfr.FeedVersionTableDelete(atx, table, id, false); err != nil {
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
	fvt := dmfr.GetFeedVersionTables()

	// Allow extension tables to not exist
	var optTables []string
	optTables = append(optTables, extraTables...)
	optTables = append(optTables, fvt.GtfsExtTables...)
	for _, table := range optTables {
		if err := dmfr.FeedVersionTableDelete(atx, table, id, true); err != nil {
			return err
		}
	}

	// Required tables
	tables := []string{}
	tables = append(tables, fvt.ImportedTables()...)
	for _, table := range tables {
		if err := dmfr.FeedVersionTableDelete(atx, table, id, false); err != nil {
			return err
		}
	}

	// Remove fvgi
	where := sq.Eq{"feed_version_id": id}
	if _, err := atx.Sqrl().Delete("feed_version_gtfs_imports").Where(where).Exec(); err != nil {
		return err
	}
	// Unset feed state
	if _, err := atx.Sqrl().Update("feed_states").Set("feed_version_id", nil).Where(where).Exec(); err != nil {
		return err
	}
	return nil
}

func DeleteFeedVersion(atx tldb.Adapter, id int, extraTables []string) error {
	// Unimport feed version first
	if err := UnimportFeedVersion(atx, id, extraTables); err != nil {
		return err
	}

	// Required tables
	fvt := dmfr.GetFeedVersionTables()
	tables := []string{}
	tables = append(tables, extraTables...)
	tables = append(tables, fvt.AllTables()...)
	for _, table := range tables {
		if err := dmfr.FeedVersionTableDelete(atx, table, id, false); err != nil {
			return err
		}
	}

	// Unset feed fetches
	where := sq.Eq{"feed_version_id": id}
	if _, err := atx.Sqrl().Update("feed_fetches").Set("feed_version_id", nil).Where(where).Exec(); err != nil {
		return err
	}
	// Delete feed version
	if _, err := atx.Sqrl().Delete("feed_versions").Where(sq.Eq{"id": id}).Exec(); err != nil {
		return err
	}
	return nil

}
