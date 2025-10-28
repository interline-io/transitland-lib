#!/bin/sh
# Set up a new postgres database and import Natural Earth data.
# set -ex -o pipefail
set -ex
SCRIPTDIR=$(dirname "$0")
TL_TEST_STORAGE=$(dirname "$0")/tmp
mkdir -p "${TL_TEST_STORAGE}"; rm -f ${TL_TEST_STORAGE}/*.zip 2>/dev/null || true

# Database names
TEST_DB="tlv2_test"
TEST_SERVER_DB="tlv2_test_server"

# Set defaults for PostgreSQL connection parameters
PGHOST=${PGHOST:-localhost}
PGPORT=${PGPORT:-5432}

# Construct auth portion of URL if credentials are provided
if [ -n "$PGUSER" ] && [ -n "$PGPASSWORD" ]; then
    DB_AUTH="${PGUSER}:${PGPASSWORD}@"
elif [ -n "$PGUSER" ]; then
    DB_AUTH="${PGUSER}@"
else
    DB_AUTH=""
fi

# Construct database URLs from individual environment variables
# This works better with Docker containers and standard PostgreSQL tooling
TL_TEST_DATABASE_URL="postgres://${DB_AUTH}${PGHOST}:${PGPORT}/${TEST_DB}?sslmode=disable"
TL_TEST_SERVER_DATABASE_URL="postgres://${DB_AUTH}${PGHOST}:${PGPORT}/${TEST_SERVER_DB}?sslmode=disable"

# Wait for database to accept connections
${SCRIPTDIR}/wait-for-it.sh "${PGHOST}:${PGPORT}"

#########################
# Rebuild binary
#########################

(cd cmd/transitland && go install .)

#########################
# DROP AND CREATE DATABASES
#########################

dropdb --if-exists $TEST_DB
createdb $TEST_DB
dropdb --if-exists $TEST_SERVER_DB
createdb $TEST_SERVER_DB

#########################
# SETUP EMPTY DATABASE
#########################

transitland dbmigrate --dburl="$TL_TEST_DATABASE_URL" up

# Load Natural Earth data
transitland dbmigrate --dburl="$TL_TEST_DATABASE_URL" natural-earth

#########################
# SETUP SERVER DATABASE
#########################

# Run migrations
transitland dbmigrate --dburl="$TL_TEST_SERVER_DATABASE_URL" up

# Load Natural Earth data
transitland dbmigrate --dburl="$TL_TEST_SERVER_DATABASE_URL" natural-earth

# Remove import files
transitland sync --dburl="$TL_TEST_SERVER_DATABASE_URL" $SCRIPTDIR/server/server-test.dmfr.json

# older data and forced error
transitland fetch --dburl="$TL_TEST_SERVER_DATABASE_URL" --storage="$TL_TEST_STORAGE" --validation-report --validation-report-storage="$TL_TEST_STORAGE" --allow-local-fetch --feed-url=$SCRIPTDIR/server/gtfs/bart-errors.zip BA # error data
transitland fetch --dburl="$TL_TEST_SERVER_DATABASE_URL" --storage="$TL_TEST_STORAGE" --validation-report --validation-report-storage="$TL_TEST_STORAGE" --allow-local-fetch --feed-url=$SCRIPTDIR/server/gtfs/bart-old.zip BA # old data
transitland import --dburl="$TL_TEST_SERVER_DATABASE_URL" --storage="$TL_TEST_STORAGE" 

# current data
transitland fetch --dburl="$TL_TEST_SERVER_DATABASE_URL" --storage="$TL_TEST_STORAGE" --validation-report --validation-report-storage="$TL_TEST_STORAGE" --allow-local-fetch 
transitland import --dburl="$TL_TEST_SERVER_DATABASE_URL" --storage="$TL_TEST_STORAGE" --activate

# sync again
transitland sync --dburl="$TL_TEST_SERVER_DATABASE_URL" $SCRIPTDIR/server/server-test.dmfr.json

# supplemental data
psql $TL_TEST_SERVER_DATABASE_URL -f $SCRIPTDIR/server/test_supplement.pgsql

# load census data
psql $TL_TEST_SERVER_DATABASE_URL -f $SCRIPTDIR/server/census/census.pgsql