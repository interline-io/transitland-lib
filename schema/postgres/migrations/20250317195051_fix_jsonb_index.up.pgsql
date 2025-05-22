BEGIN;

--- feeds
drop index index_current_feeds_on_auth;
drop index index_current_feeds_on_urls;
drop index current_feeds_feed_tags_idx;
create index on current_feeds using gin(auth);
create index on current_feeds using gin(urls);
create index on current_feeds using gin(feed_tags);
create index on current_feeds using gin(license);

--- operators
drop index current_operators_operator_tags_idx;
create index on current_operators using gin(associated_feeds);
create index on current_operators using gin(operator_tags);

COMMIT;