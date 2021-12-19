BEGIN;

alter table feed_version_gtfs_imports add column schedule_removed bool not null default false;

COMMIT;