BEGIN;

alter table feed_fetches add column response_ttfb_ms integer;
alter table feed_fetches add column response_time_ms integer;

COMMIT;