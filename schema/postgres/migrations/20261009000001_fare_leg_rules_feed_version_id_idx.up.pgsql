-- Supports reading a feed version's fare leg rules and join rules in id order;
-- without it, Postgres walks the primary key across every feed version.
-- 20260317000610_feed_version_id_idx added this index to the other fares tables.
BEGIN;

CREATE INDEX IF NOT EXISTS gtfs_fare_leg_rules_feed_version_id_id_idx ON gtfs_fare_leg_rules (feed_version_id, id);
CREATE INDEX IF NOT EXISTS gtfs_fare_leg_join_rules_feed_version_id_id_idx ON gtfs_fare_leg_join_rules (feed_version_id, id);

COMMIT;
