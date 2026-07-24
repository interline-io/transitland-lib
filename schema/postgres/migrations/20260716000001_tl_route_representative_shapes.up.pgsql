BEGIN;

-- Pointers from a route to the gtfs_shapes chosen to represent it.
-- Rank 0 is the route's primary shape; the full set spans its combined geometry.
-- direction_id is the direction the shape was selected for.
-- The metrics are per shape; tl_route_geometries holds the route-level aggregates
-- (max of length, max_segment_length, first_point_max_distance; any of generated).
CREATE TABLE public.tl_route_representative_shapes (
    feed_version_id bigint NOT NULL,
    route_id bigint NOT NULL,
    shape_id bigint NOT NULL,
    direction_id integer,
    rank integer NOT NULL,
    generated boolean NOT NULL,
    length double precision,
    max_segment_length double precision,
    first_point_max_distance double precision
);

ALTER TABLE ONLY public.tl_route_representative_shapes
    ADD CONSTRAINT tl_route_representative_shapes_feed_version_id_fkey
    FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);

ALTER TABLE ONLY public.tl_route_representative_shapes
    ADD CONSTRAINT tl_route_representative_shapes_route_id_fkey
    FOREIGN KEY (route_id) REFERENCES public.gtfs_routes(id);

ALTER TABLE ONLY public.tl_route_representative_shapes
    ADD CONSTRAINT tl_route_representative_shapes_shape_id_fkey
    FOREIGN KEY (shape_id) REFERENCES public.gtfs_shapes(id);

CREATE UNIQUE INDEX tl_route_representative_shapes_route_id_rank_idx
    ON public.tl_route_representative_shapes (route_id, rank);

CREATE INDEX tl_route_representative_shapes_route_id_direction_id_idx
    ON public.tl_route_representative_shapes (route_id, direction_id);

CREATE INDEX tl_route_representative_shapes_feed_version_id_idx
    ON public.tl_route_representative_shapes (feed_version_id);

CREATE INDEX tl_route_representative_shapes_shape_id_idx
    ON public.tl_route_representative_shapes (shape_id);

COMMIT;
