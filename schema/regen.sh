set -ex
echo "CREATE EXTENSION postgis;" > postgres.pgsql
echo "CREATE EXTENSION hstore;" >> postgres.pgsql
pg_dump \
    -t 'current_feeds' \
    -t 'feed_versions' \
    -t 'feed_version_gtfs_imports' \
    -t 'feed_version_geometries' \
    -t 'agency_geometries' \
    -t 'route_geometries' \
    -t 'route_stops' \
    -t 'active_*' \
    -t 'feed_states' \
    -t 'gtfs_*' \
    --no-owner \
    -s \
    --no-comments $PGDATABASE | egrep -v "^(SET|SELECT pg_catalog|--)" | sed -e '/^$/d'  >> postgres.pgsql

# rails compat
pg_dump -t 'schema_migrations' --inserts --no-owner --no-comments $PGDATABASE >> postgres.pgsql

(cd ../internal; statik -src=../schema -p schema)
