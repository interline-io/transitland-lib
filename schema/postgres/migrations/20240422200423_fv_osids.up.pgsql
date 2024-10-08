BEGIN;

CREATE TABLE feed_version_agency_onestop_ids (
    feed_version_id bigint REFERENCES feed_versions(id) NOT NULL,
    entity_id text not null,
    onestop_id text not null
);
CREATE UNIQUE INDEX ON feed_version_agency_onestop_ids(feed_version_id, entity_id);
CREATE INDEX ON feed_version_agency_onestop_ids(onestop_id) INCLUDE (entity_id, feed_version_id);

CREATE TABLE feed_version_route_onestop_ids (
    feed_version_id bigint REFERENCES feed_versions(id) NOT NULL,
    entity_id text not null,
    onestop_id text not null
);
CREATE UNIQUE INDEX ON feed_version_route_onestop_ids(feed_version_id, entity_id);
CREATE INDEX ON feed_version_route_onestop_ids(onestop_id) INCLUDE (entity_id, feed_version_id);

CREATE TABLE feed_version_stop_onestop_ids (
    feed_version_id bigint REFERENCES feed_versions(id) NOT NULL,
    entity_id text not null,
    onestop_id text not null
);
CREATE UNIQUE INDEX ON feed_version_stop_onestop_ids(feed_version_id, entity_id);
CREATE INDEX ON feed_version_stop_onestop_ids(onestop_id) INCLUDE (entity_id, feed_version_id);

COMMIT;
