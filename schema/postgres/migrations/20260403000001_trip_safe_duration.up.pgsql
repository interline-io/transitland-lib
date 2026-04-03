-- Add GTFS-Flex safe duration fields to trips table
-- See: https://github.com/google/transit/pull/598
ALTER TABLE public.gtfs_trips
    ADD COLUMN safe_duration_factor double precision,
    ADD COLUMN safe_duration_offset double precision;
