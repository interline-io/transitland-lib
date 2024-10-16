BEGIN;

alter table gtfs_frequencies alter column exact_times drop not null;
alter table gtfs_fare_attributes alter column transfer_duration drop not null;

-------------

alter table gtfs_fare_rules alter column origin_id drop not null;
alter table gtfs_fare_rules alter column destination_id drop not null;
alter table gtfs_fare_rules alter column contains_id drop not null;

-------------

alter table gtfs_agencies alter column agency_lang drop not null;
alter table gtfs_agencies alter column agency_phone drop not null;
alter table gtfs_agencies alter column agency_fare_url drop not null;
alter table gtfs_agencies alter column agency_email drop not null;

-------------

alter table gtfs_routes alter column route_short_name drop not null;
alter table gtfs_routes alter column route_long_name drop not null;
alter table gtfs_routes alter column route_desc drop not null;
alter table gtfs_routes alter column route_url drop not null;
alter table gtfs_routes alter column route_color drop not null;
alter table gtfs_routes alter column route_text_color drop not null;
alter table gtfs_routes alter column route_sort_order drop not null;

-------------

alter table gtfs_trips alter column trip_short_name drop not null;
alter table gtfs_trips alter column trip_headsign drop not null;
alter table gtfs_trips alter column block_id drop not null;
alter table gtfs_trips alter column wheelchair_accessible drop not null;
alter table gtfs_trips alter column bikes_allowed drop not null;
alter table gtfs_trips alter column journey_pattern_id drop not null;
-- DO NOT DROP NOT NULL for stop_pattern_id, journey_pattern_offset

-------------

alter table gtfs_stops alter column stop_code drop not null;
alter table gtfs_stops alter column stop_name drop not null;
alter table gtfs_stops alter column stop_desc drop not null;
alter table gtfs_stops alter column zone_id drop not null;
alter table gtfs_stops alter column stop_url drop not null;
alter table gtfs_stops alter column stop_timezone drop not null;
alter table gtfs_stops alter column wheelchair_boarding drop not null;
-- DO NOT DROP NOT NULL for location_type

------------

alter table gtfs_feed_infos alter column feed_version_name drop not null;

alter table gtfs_levels alter column level_name drop not null;

alter table gtfs_pathways alter column length drop not null;
alter table gtfs_pathways alter column traversal_time drop not null;
alter table gtfs_pathways alter column stair_count drop not null;
alter table gtfs_pathways alter column max_slope drop not null;
alter table gtfs_pathways alter column min_width drop not null;
alter table gtfs_pathways alter column signposted_as drop not null;
alter table gtfs_pathways alter column reverse_signposted_as drop not null;

alter table gtfs_transfers alter column from_stop_id drop not null;
alter table gtfs_transfers alter column to_stop_id drop not null;

COMMIT;
