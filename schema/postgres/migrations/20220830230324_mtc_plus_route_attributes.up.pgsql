BEGIN;

CREATE TABLE ext_plus_route_attributes (
    id bigserial primary key not null,
    feed_version_id bigint REFERENCES feed_versions(id),
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,

    route_id bigint not null,
    category integer,
    subcategory integer,
    running_way integer
);
CREATE INDEX ON ext_plus_route_attributes(feed_version_id);
CREATE INDEX ON ext_plus_route_attributes(route_id);

COMMIT;