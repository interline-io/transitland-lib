BEGIN;

alter table feed_fetches add column validation_duration_ms integer;
alter table feed_fetches add column upload_duration_ms integer;

COMMIT;