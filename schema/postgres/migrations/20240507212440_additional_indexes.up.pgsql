BEGIN;

create index on feed_states(feed_id) include (feed_version_id,public);
create index on feed_versions(feed_id) include (id,sha1);
create unique index on tl_route_geometries(route_id);

COMMIT;