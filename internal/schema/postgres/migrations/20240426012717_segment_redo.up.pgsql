BEGIN;

CREATE TABLE tl_segments (
    id bigserial primary key not null,
    feed_version_id bigint references feed_versions(id) not null,
    geometry geography(LineString,4326) not null,
    way_id text not null
);
CREATE INDEX ON tl_segments(feed_version_id);

CREATE TABLE tl_segment_patterns (
    id bigserial primary key not null,
    feed_version_id bigint references feed_versions(id) not null,
    segment_id bigint references tl_segments(id) not null,
    route_id bigint references gtfs_routes(id) not null,    
    shape_id bigint references gtfs_shapes(id) not null,
    stop_pattern_id int not null
);
CREATE UNIQUE INDEX ON tl_segment_patterns(segment_id,route_id,shape_id,stop_pattern_id);
CREATE INDEX ON tl_segment_patterns(route_id);

COMMIT;