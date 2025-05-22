package dmfr

type FeedVersionTables struct {
	FetchStatDerivedTables []string
	ImportDerivedTables    []string
	SystemTables           []string
	GtfsAnonTables         []string
	GtfsEntityTables       []string
	GtfsExtTables          []string
}

func (t FeedVersionTables) AllTables() []string {
	var ret []string
	ret = append(ret, t.ImportDerivedTables...)
	ret = append(ret, t.GtfsAnonTables...)
	ret = append(ret, t.GtfsEntityTables...)
	ret = append(ret, t.FetchStatDerivedTables...)
	ret = append(ret, t.SystemTables...)
	return ret
}

func (t FeedVersionTables) ImportedTables() []string {
	var ret []string
	ret = append(ret, t.ImportDerivedTables...)
	ret = append(ret, t.GtfsAnonTables...)
	ret = append(ret, t.GtfsEntityTables...)
	return ret
}

func (t FeedVersionTables) ScheduleTables() []string {
	return []string{
		"gtfs_stop_times",
		"gtfs_transfers",
		"gtfs_calendar_dates",
		"gtfs_frequencies",
		"gtfs_trips",
		"gtfs_shapes",
		"gtfs_calendars",
	}
}

func GetFeedVersionTables() FeedVersionTables {
	// Set of tables to delete where feed_version_id = fvid
	// Table order is very important!
	return FeedVersionTables{
		FetchStatDerivedTables: []string{
			"feed_version_file_infos",
			"feed_version_service_levels",
			"feed_version_service_windows",
			"feed_version_agency_onestop_ids",
			"feed_version_route_onestop_ids",
			"feed_version_stop_onestop_ids",
		},
		ImportDerivedTables: []string{
			"tl_feed_version_geometries",
			"tl_route_headways",
			"tl_agency_places",
			"tl_route_stops",
			"tl_agency_geometries",
			"tl_route_geometries",
			"tl_agency_onestop_ids",
			"tl_route_onestop_ids",
			"tl_stop_onestop_ids",
		},
		GtfsExtTables: []string{
			"tl_stop_external_references",
			"tl_ext_fare_networks",
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
			"ext_plus_route_attributes",
		},
		GtfsAnonTables: []string{
			"gtfs_stop_times",
			"gtfs_stop_areas",
			"gtfs_transfers",
			"gtfs_calendar_dates",
			"gtfs_feed_infos",
			"gtfs_frequencies",
			"gtfs_fare_rules",
			"gtfs_attributions",
			"gtfs_translations",
		},
		GtfsEntityTables: []string{
			"gtfs_fare_media",
			"gtfs_fare_leg_rules",
			"gtfs_fare_products",
			"gtfs_fare_transfer_rules",
			"gtfs_rider_categories",
			"gtfs_route_networks",
			"gtfs_networks",
			"gtfs_timeframes",
			"gtfs_areas",
			"gtfs_pathways",
			"gtfs_fare_attributes",
			"gtfs_trips",
			"gtfs_shapes",
			"gtfs_calendars",
			"gtfs_stops",
			"gtfs_levels",
			"gtfs_routes",
			"gtfs_agencies",
		},
		SystemTables: []string{
			"feed_version_gtfs_imports",
			"tl_validation_reports",
		},
	}
}
