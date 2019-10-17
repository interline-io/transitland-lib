echo "CREATE EXTENSION postgis;" > postgres.pgsql
echo "CREATE EXTENSION hstore;" >> postgres.pgsql
pg_dump -t 'current_feeds' -t 'feed_versions' -t 'gtfs_*' -s $DB --no-owner --no-comments | egrep -v "^(SET|SELECT pg_catalog|--)" | sed -e '/^$/d'  >> postgres.pgsql
(cd ../internal; statik -src=../schema)