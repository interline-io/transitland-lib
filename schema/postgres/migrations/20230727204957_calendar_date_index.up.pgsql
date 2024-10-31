BEGIN;

CREATE INDEX ON gtfs_calendar_dates(feed_version_id,date,exception_type);
CREATE INDEX current_feeds_id_deleted_at_idx ON public.current_feeds USING btree (id) INCLUDE (deleted_at);
CREATE INDEX tl_route_headways_route_id_idx ON public.tl_route_headways USING btree (route_id);
ALTER TABLE feed_states ALTER COLUMN fetch_wait DROP NOT NULL;
ALTER TABLE feed_states ALTER COLUMN fetch_wait DROP DEFAULT;

COMMIT;
