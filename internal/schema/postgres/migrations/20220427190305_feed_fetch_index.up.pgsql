BEGIN;

create index on feed_fetches(feed_id,fetched_at) include(success);

COMMIT;