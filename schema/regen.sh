set -ex
echo "CREATE EXTENSION postgis;" > postgres.pgsql
echo "CREATE EXTENSION hstore;" >> postgres.pgsql
pg_dump \
    -t 'current_feeds' \
    -t 'feed_versions' \
    -t 'feed_version_gtfs_imports' \
    -t 'feed_states' \
    -t 'gtfs_*' \
    --no-owner \
    -s \
    --no-comments $DB | egrep -v "^(SET|SELECT pg_catalog|--)" | sed -e '/^$/d'  >> postgres.pgsql

# rails compat
pg_dump -t 'schema_migrations' --no-owner --no-comments $DB >> postgres.pgsql

(cd ../internal; statik -src=../schema -p schema)
