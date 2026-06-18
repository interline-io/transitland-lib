-- Track whether a feed version import was initiated automatically (by a
-- maintenance/queue process) or manually (by a user). Used by garbage
-- collection to retain user-initiated imports longer. Existing rows predate
-- the feature and are treated as automatic.
ALTER TABLE feed_version_gtfs_imports
  ADD COLUMN import_source text NOT NULL DEFAULT 'automatic';
