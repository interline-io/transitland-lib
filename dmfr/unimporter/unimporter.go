package unimporter

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/tldb"
)

type Options struct {
	FeedVersionID int
	ExtraTables   []string
}

type Result struct {
	Success      bool
	ExceptionLog string
}

// UnimportFeedVersion
func UnimportFeedVersion(atx tldb.Adapter, opts Options) (Result, error) {
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
		"gtfs_pathways",
		// named entities
		"gtfs_fare_rules",
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
	err := atx.Tx(func(model tldb.Adapter) error {
		id := opts.FeedVersionID
		where := sq.Eq{"feed_version_id": id}
		for _, table := range opts.ExtraTables {
			_, err := model.Sqrl().Delete(table).Where(where).Exec()
			if err != nil {
				return err
			}
		}
		for _, table := range tables {
			_, err := model.Sqrl().Delete(table).Where(where).Exec()
			if err != nil {
				return err
			}
		}
		if _, err := model.Sqrl().Delete("feed_version_gtfs_imports").Where(where).Exec(); err != nil {
			return err
		}
		if _, err := model.Sqrl().Update("feed_states").Set("feed_version_id", nil).Where(where).Exec(); err != nil {
			return err
		}
		return nil
	})
	res := Result{}
	if err != nil {
		return res, err
	}
	return res, nil
}
