BEGIN;

ALTER TABLE tl_stop_external_references RENAME TO old_tl_stop_external_references;

CREATE TABLE tl_stop_external_references (
    id bigserial primary key not null,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id int not null references feed_versions(id),
    stop_id bigint not null references gtfs_stops(id),
    target_feed_onestop_id text,
    target_stop_id text,
    inactive boolean
);
CREATE INDEX ON tl_stop_external_references (feed_version_id);
CREATE INDEX ON tl_stop_external_references (stop_id);

insert into tl_stop_external_references(feed_version_id, stop_id, target_feed_onestop_id, target_stop_id, inactive, created_at, updated_at) select feed_version_id, id as stop_id, target_feed_onestop_id, target_stop_id, inactive, created_at, updated_at from old_tl_stop_external_references ;
drop table old_tl_stop_external_references;

COMMIT;
