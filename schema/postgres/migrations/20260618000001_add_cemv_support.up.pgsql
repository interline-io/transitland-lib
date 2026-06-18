BEGIN;

-- Add cemv_support column to the raw and materialized agency/route tables
ALTER TABLE public.gtfs_agencies ADD COLUMN cemv_support integer;
ALTER TABLE public.gtfs_routes ADD COLUMN cemv_support integer;
ALTER TABLE public.tl_materialized_active_agencies ADD COLUMN cemv_support integer;
ALTER TABLE public.tl_materialized_active_routes ADD COLUMN cemv_support integer;

COMMIT;
