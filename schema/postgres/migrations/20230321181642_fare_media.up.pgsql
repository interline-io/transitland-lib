BEGIN;

DROP TABLE ext_faresv2_areas               ;
DROP TABLE ext_faresv2_fare_capping        ;
DROP TABLE ext_faresv2_fare_containers     ;
DROP TABLE ext_faresv2_fare_leg_rules      ;
DROP TABLE ext_faresv2_fare_products       ;
DROP TABLE ext_faresv2_fare_timeframes     ;
DROP TABLE ext_faresv2_fare_transfer_rules ;
DROP TABLE ext_faresv2_rider_categories    ;

ALTER TABLE gtfs_fare_containers RENAME TO gtfs_fare_media;
ALTER TABLE gtfs_fare_media RENAME COLUMN fare_container_id TO fare_media_id;
ALTER TABLE gtfs_fare_media RENAME COLUMN fare_container_name TO fare_media_name;
ALTER TABLE gtfs_fare_media ADD COLUMN fare_media_type integer;
ALTER TABLE gtfs_fare_media DROP COLUMN minimum_initial_purchase;
ALTER TABLE gtfs_fare_media DROP COLUMN amount;
ALTER TABLE gtfs_fare_media DROP COLUMN currency;

ALTER TABLE gtfs_fare_products RENAME COLUMN fare_container_id TO fare_media_id;

ALTER TABLE gtfs_fare_media RENAME CONSTRAINT gtfs_fare_containers_feed_version_id_fkey TO gtfs_fare_media_feed_version_id_fkey;

ALTER INDEX gtfs_fare_containers_pkey RENAME TO gtfs_fare_media_pkey;
ALTER INDEX gtfs_fare_containers_feed_version_id_fare_container_id_idx RENAME TO gtfs_fare_media_feed_version_id_fare_media_id_idx;
ALTER INDEX gtfs_fare_containers_feed_version_id_idx RENAME TO gtfs_fare_media_feed_version_id_idx;

ALTER SEQUENCE gtfs_fare_containers_id_seq RENAME TO gtfs_fare_media_id_seq;

update gtfs_fare_media set fare_media_type = 0 where fare_media_id = 'cash';
update gtfs_fare_media set fare_media_type = 2 where fare_media_id = 'clipper';
update gtfs_fare_media set fare_media_type = 4 where fare_media_id = 'ezfare';
update gtfs_fare_media set fare_media_type = 4 where fare_media_id = 'munimobile';
update gtfs_fare_media set fare_media_type = 4 where fare_media_id = 'tokentransit';

COMMIT;