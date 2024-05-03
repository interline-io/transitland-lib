BEGIN;

create index on feed_versions(feed_id);
create index on feed_versions(id) include (feed_id,sha1);
create index on current_feeds(id) include (deleted_at,onestop_id);

drop index index_gtfs_calendar_dates_on_feed_version_id;
drop index gtfs_areas_feed_version_id_idx;
drop index gtfs_fare_leg_rules_feed_version_id_idx;
drop index gtfs_fare_media_feed_version_id_idx;
drop index gtfs_fare_products_feed_version_id_idx;
drop index gtfs_fare_transfer_rules_feed_version_id_idx;
drop index gtfs_stop_areas_area_id_idx;

alter table feed_states drop column last_fetched_at;
alter table feed_states drop column last_successful_fetch_at;
alter table feed_states drop column last_fetch_error;
alter table feed_states drop column feed_realtime_enabled;
alter table feed_states drop column tags;
-- alter table feed_states drop column feed_priority;
-- alter table feed_states drop column created_at;
-- alter table feed_states drop column updated_at;

alter table current_feeds drop column last_fetch_error;
alter table current_feeds drop column last_successful_fetch_at;
alter table current_feeds drop column edited_attributes;
alter table current_feeds drop column active_feed_version_id;
alter table current_feeds drop column geometry;
alter table current_feeds drop column created_or_updated_in_changeset_id;
alter table current_feeds drop column version;
alter table current_feeds drop column last_fetched_at;
alter table current_feeds drop column last_imported_at;

alter table feed_versions drop column file_feedvalidator;
alter table feed_versions drop column import_level;
alter table feed_versions drop column imported_at;
alter table feed_versions drop column md5;
alter table feed_versions drop column md5_raw;
alter table feed_versions drop column sha1_raw;
alter table feed_versions drop column file_raw;

COMMIT;
