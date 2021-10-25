set -ex

echo "CREATE EXTENSION postgis; CREATE EXTENSION hstore;" > postgres.pgsql
echo "CREATE EXTENSION pg_trgm; CREATE EXTENSION unaccent; CREATE TEXT SEARCH CONFIGURATION tl ( COPY = simple ); ALTER TEXT SEARCH CONFIGURATION tl ALTER MAPPING FOR hword, hword_part, word WITH unaccent;" >> postgres.pgsql

pg_dump \
    -t 'current_feeds' \
    -t 'feed_versions' \
    -t 'feed_version_gtfs_imports' \
    -t 'feed_version_file_infos' \
    -t 'feed_version_service_levels' \
    -t 'feed_states' \
    -t "current_operators" \
    -t "current_operators_in_feed" \
    -t 'gtfs_*' \
    -t "ext_*" \
    -t "tl_agency_geometries" \
    -t "tl_agency_onestop_ids" \
    -t "tl_agency_places" \
    -t "tl_feed_version_geometries" \
    -t "tl_route_geometries" \
    -t "tl_route_headways" \
    -t "tl_route_onestop_ids" \
    -t "tl_route_stops" \
    -t "tl_stop_onestop_ids" \
    -t "tl_census_*" \
    -t "tl_stop_external_references" \
    -t "tl_ext_fare_networks" \
    -t "tl_ext_gtfs_stops" \
    -t "ne_10m_admin_1_states_provinces" \
    -t "ne_10m_populated_places" \
    -s \
    --no-owner \
    --no-comments $PGDATABASE | egrep -v "^(SET|SELECT pg_catalog|--)" | sed -e '/^$/d'  >> postgres.pgsql
