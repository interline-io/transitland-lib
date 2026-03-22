BEGIN;

-- Already has (feed_version_id, id) index: gtfs_stops (gtfs_stops_feed_version_id_id_idx)
-- Already has (feed_version_id, id, ...) index: gtfs_routes (index_gtfs_routes_on_feed_version_id_agency_id)

-- Tables without an id column are excluded: feed_version_agency_onestop_ids, feed_version_route_onestop_ids,
--   feed_version_stop_onestop_ids, gtfs_stop_times, tl_agency_geometries, tl_feed_version_geometries,
--   tl_route_geometries, tl_route_stops

-- Not needed, we don't order by (feed_version_id,id) in application code:
--  feed_fetches
--  feed_states
--  feed_version_gtfs_imports
--  tl_stop_external_references
--  tl_stop_onestop_ids
--  tl_ext_fare_networks
--  tl_agency_onestop_ids
--  tl_route_headways
--  tl_route_onestop_ids
--  tl_validation_reports

-- Run these first, they are by far the largest tables
CREATE INDEX IF NOT EXISTS gtfs_trips_feed_version_id_id_idx ON gtfs_trips (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_shapes_feed_version_id_id_idx ON gtfs_shapes (feed_version_id, id);

-- GTFS entity tables
CREATE INDEX IF NOT EXISTS ext_plus_calendar_attributes_feed_version_id_id_idx ON ext_plus_calendar_attributes (feed_version_id, id);
CREATE INDEX IF NOT EXISTS ext_plus_directions_feed_version_id_id_idx ON ext_plus_directions (feed_version_id, id);
CREATE INDEX IF NOT EXISTS ext_plus_fare_rider_categories_feed_version_id_id_idx ON ext_plus_fare_rider_categories (feed_version_id, id);
CREATE INDEX IF NOT EXISTS ext_plus_farezone_attributes_feed_version_id_id_idx ON ext_plus_farezone_attributes (feed_version_id, id);
CREATE INDEX IF NOT EXISTS ext_plus_realtime_routes_feed_version_id_id_idx ON ext_plus_realtime_routes (feed_version_id, id);
CREATE INDEX IF NOT EXISTS ext_plus_realtime_stops_feed_version_id_id_idx ON ext_plus_realtime_stops (feed_version_id, id);
CREATE INDEX IF NOT EXISTS ext_plus_realtime_trips_feed_version_id_id_idx ON ext_plus_realtime_trips (feed_version_id, id);
CREATE INDEX IF NOT EXISTS ext_plus_rider_categories_feed_version_id_id_idx ON ext_plus_rider_categories (feed_version_id, id);
CREATE INDEX IF NOT EXISTS ext_plus_route_attributes_feed_version_id_id_idx ON ext_plus_route_attributes (feed_version_id, id);
CREATE INDEX IF NOT EXISTS ext_plus_stop_attributes_feed_version_id_id_idx ON ext_plus_stop_attributes (feed_version_id, id);
CREATE INDEX IF NOT EXISTS ext_plus_timepoints_feed_version_id_id_idx ON ext_plus_timepoints (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_agencies_feed_version_id_id_idx ON gtfs_agencies (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_areas_feed_version_id_id_idx ON gtfs_areas (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_attributions_feed_version_id_id_idx ON gtfs_attributions (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_booking_rules_feed_version_id_id_idx ON gtfs_booking_rules (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_calendar_dates_feed_version_id_id_idx ON gtfs_calendar_dates (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_calendars_feed_version_id_id_idx ON gtfs_calendars (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_fare_attributes_feed_version_id_id_idx ON gtfs_fare_attributes (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_fare_media_feed_version_id_id_idx ON gtfs_fare_media (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_fare_products_feed_version_id_id_idx ON gtfs_fare_products (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_fare_rules_feed_version_id_id_idx ON gtfs_fare_rules (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_fare_transfer_rules_feed_version_id_id_idx ON gtfs_fare_transfer_rules (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_feed_infos_feed_version_id_id_idx ON gtfs_feed_infos (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_frequencies_feed_version_id_id_idx ON gtfs_frequencies (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_levels_feed_version_id_id_idx ON gtfs_levels (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_location_group_stops_feed_version_id_id_idx ON gtfs_location_group_stops (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_location_groups_feed_version_id_id_idx ON gtfs_location_groups (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_locations_feed_version_id_id_idx ON gtfs_locations (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_networks_feed_version_id_id_idx ON gtfs_networks (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_pathways_feed_version_id_id_idx ON gtfs_pathways (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_rider_categories_feed_version_id_id_idx ON gtfs_rider_categories (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_route_networks_feed_version_id_id_idx ON gtfs_route_networks (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_stop_areas_feed_version_id_id_idx ON gtfs_stop_areas (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_timeframes_feed_version_id_id_idx ON gtfs_timeframes (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_transfers_feed_version_id_id_idx ON gtfs_transfers (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_translations_feed_version_id_id_idx ON gtfs_translations (feed_version_id, id);

-- Materialized tables
CREATE INDEX IF NOT EXISTS tl_materialized_active_agencies_feed_version_id_id_idx ON tl_materialized_active_agencies (feed_version_id, id);
CREATE INDEX IF NOT EXISTS tl_materialized_active_routes_feed_version_id_id_idx ON tl_materialized_active_routes (feed_version_id, id);
CREATE INDEX IF NOT EXISTS tl_materialized_active_stops_feed_version_id_id_idx ON tl_materialized_active_stops (feed_version_id, id);

-- Segments
CREATE INDEX IF NOT EXISTS tl_segment_patterns_feed_version_id_id_idx ON tl_segment_patterns (feed_version_id, id);
CREATE INDEX IF NOT EXISTS tl_segments_feed_version_id_id_idx ON tl_segments (feed_version_id, id);

-- Other feed version tables
CREATE INDEX IF NOT EXISTS feed_version_file_infos_feed_version_id_id_idx ON feed_version_file_infos (feed_version_id, id);
CREATE INDEX IF NOT EXISTS feed_version_service_levels_feed_version_id_id_idx ON feed_version_service_levels (feed_version_id, id);
CREATE INDEX IF NOT EXISTS feed_version_service_windows_feed_version_id_id_idx ON feed_version_service_windows (feed_version_id, id);
CREATE INDEX IF NOT EXISTS tl_agency_places_feed_version_id_id_idx ON tl_agency_places (feed_version_id, id);


COMMIT;
