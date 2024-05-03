BEGIN;

ALTER TABLE feed_version_file_infos ADD COLUMN values_count jsonb not null default '{}'::jsonb;
ALTER TABLE feed_version_file_infos ADD COLUMN values_unique jsonb not null default '{}'::jsonb;

COMMIT;