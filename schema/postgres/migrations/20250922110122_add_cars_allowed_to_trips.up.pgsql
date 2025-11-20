-- Add cars_allowed column to gtfs_trips table
ALTER TABLE public.gtfs_trips ADD COLUMN cars_allowed integer;
