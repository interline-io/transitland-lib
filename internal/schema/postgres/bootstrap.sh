#!/bin/sh
# Set up a new postgres database and import Natural Earth data.
set -e
SCRIPTDIR=$(dirname "$0")

# Wait for database to accept connections
${SCRIPTDIR}/wait-for-it.sh "$PGHOST:$PGPORT"

# Fail if db already exists -- this is a bootstrap script after all!
createdb "$PGDATABASE"

# Database schema
migrate -path="${SCRIPTDIR}/migrations" -database="postgres://$PGUSER:$PGPASSWORD@$PGHOST:$PGPORT/$PGDATABASE?sslmode=disable" up

# Load Natural Earth
UNZIPDIR=$(mktemp -d)
DATADIR="${SCRIPTDIR}/../ne"
(cd "$UNZIPDIR" && unzip "${DATADIR}/ne_10m_admin_1_states_provinces.zip")
(cd "$UNZIPDIR" && unzip "${DATADIR}/ne_10m_populated_places_simple.zip")
ogr2ogr -f "PostgreSQL" PG:"" $UNZIPDIR/ne_10m_populated_places_simple.shp -nln ne_10m_populated_places -lco GEOM_TYPE=geography -lco GEOMETRY_NAME=geometry -overwrite
ogr2ogr -f "PostgreSQL" PG:"" $UNZIPDIR/ne_10m_admin_1_states_provinces.shp -nln ne_10m_admin_1_states_provinces -lco GEOM_TYPE=geography -lco GEOMETRY_NAME=geometry -nlt PROMOTE_TO_MULTI  -overwrite
rm ${UNZIPDIR}/*
rmdir ${UNZIPDIR}