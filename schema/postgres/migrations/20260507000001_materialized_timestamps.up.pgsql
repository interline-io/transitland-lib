BEGIN;

ALTER TABLE tl_materialized_active_stops
    ADD COLUMN created_at timestamp without time zone DEFAULT now() NOT NULL,
    ADD COLUMN updated_at timestamp without time zone DEFAULT now() NOT NULL;

ALTER TABLE tl_materialized_active_routes
    ADD COLUMN created_at timestamp without time zone DEFAULT now() NOT NULL,
    ADD COLUMN updated_at timestamp without time zone DEFAULT now() NOT NULL;

ALTER TABLE tl_materialized_active_agencies
    ADD COLUMN created_at timestamp without time zone DEFAULT now() NOT NULL,
    ADD COLUMN updated_at timestamp without time zone DEFAULT now() NOT NULL;

UPDATE tl_materialized_active_stops m
SET created_at = s.created_at,
    updated_at = s.updated_at
FROM gtfs_stops s
WHERE s.id = m.id;

UPDATE tl_materialized_active_routes m
SET created_at = r.created_at,
    updated_at = r.updated_at
FROM gtfs_routes r
WHERE r.id = m.id;

UPDATE tl_materialized_active_agencies m
SET created_at = a.created_at,
    updated_at = a.updated_at
FROM gtfs_agencies a
WHERE a.id = m.id;

COMMIT;
