#!/bin/sh
# Set up a new postgres database and import Natural Earth data.
set -ex
SCRIPTDIR=$(dirname "$0")
TL_TEST_STORAGE=$(dirname "$0")/tmp
mkdir -p "${TL_TEST_STORAGE}"; rm -f ${TL_TEST_STORAGE}/*.zip 2>/dev/null || true

# Validate required environment variables
if [ -z "$TL_TEST_DATABASE_URL" ]; then
    echo "Error: TL_TEST_DATABASE_URL must be set"
    exit 1
fi
if [ -z "$TL_TEST_SERVER_DATABASE_URL" ]; then
    echo "Error: TL_TEST_SERVER_DATABASE_URL must be set"
    exit 1
fi

# Function to parse PostgreSQL connection parameters from URL
# Format: postgres://[user:pass@]host:port/dbname?params
parse_pg_url() {
    local url="$1"
    if echo "$url" | grep -q "@"; then
        # URL has user:pass@ - extract all components
        export PGUSER=$(echo "$url" | awk -F'[/:]' '{print $3}')
        export PGPASSWORD=$(echo "$url" | awk -F'[:@]' '{print $4}')
        export PGHOST=$(echo "$url" | awk -F'[@/]' '{print $4}' | awk -F':' '{print $1}')
        export PGPORT=$(echo "$url" | awk -F'[@/]' '{print $4}' | awk -F':' '{print $2}')
        export PGDATABASE=$(echo "$url" | awk -F'[@/]' '{print $5}' | awk -F'[?]' '{print $1}')
    else
        # URL has no auth - extract host:port:dbname after postgres://
        export PGHOST=$(echo "$url" | awk -F'[/:]' '{print $4}')
        export PGPORT=$(echo "$url" | awk -F'[/:]' '{print $5}')
        export PGDATABASE=$(echo "$url" | awk -F'[/:]' '{print $6}' | awk -F'[?]' '{print $1}')
    fi
}

#########################
# Rebuild binary
#########################

(cd cmd/transitland && go install .)

#########################
# Migrate and init base database
#########################

# Parse connection parameters from TL_TEST_DATABASE_URL
parse_pg_url "$TL_TEST_DATABASE_URL"
echo "PGHOST=$PGHOST PGPORT=$PGPORT PGDATABASE=$PGDATABASE PGUSER=$PGUSER PGPASSWORD=$PGPASSWORD"
"${SCRIPTDIR}/wait-for-it.sh" -h "$PGHOST" -p "$PGPORT" -t 30

# Drop and recreate database
dropdb --if-exists "$PGDATABASE"
createdb "$PGDATABASE"

# Run migrations
transitland dbmigrate --dburl="$TL_TEST_DATABASE_URL" up
transitland dbmigrate --dburl="$TL_TEST_DATABASE_URL" natural-earth

#########################
# Migrate and init server database
#########################

# Extract database names from URLs for backwards compatibility
parse_pg_url "$TL_TEST_SERVER_DATABASE_URL"
echo "PGHOST=$PGHOST PGPORT=$PGPORT PGDATABASE=$PGDATABASE PGUSER=$PGUSER PGPASSWORD=$PGPASSWORD"
"${SCRIPTDIR}/wait-for-it.sh" -h "$PGHOST" -p "$PGPORT" -t 30

# Drop and recreate database
dropdb --if-exists "$PGDATABASE"
createdb "$PGDATABASE"

# Run migrations
transitland dbmigrate --dburl="$TL_TEST_SERVER_DATABASE_URL" up
transitland dbmigrate --dburl="$TL_TEST_SERVER_DATABASE_URL" natural-earth

# Remove import files
transitland sync --dburl="$TL_TEST_SERVER_DATABASE_URL" "$SCRIPTDIR/server/server-test.dmfr.json"

# Older data and forced error
transitland fetch --dburl="$TL_TEST_SERVER_DATABASE_URL" --storage="$TL_TEST_STORAGE" --validation-report --validation-report-storage="$TL_TEST_STORAGE" --allow-local-fetch --feed-url="$SCRIPTDIR/server/gtfs/bart-errors.zip" BA # error data
transitland fetch --dburl="$TL_TEST_SERVER_DATABASE_URL" --storage="$TL_TEST_STORAGE" --validation-report --validation-report-storage="$TL_TEST_STORAGE" --allow-local-fetch --feed-url="$SCRIPTDIR/server/gtfs/bart-old.zip" BA # old data
transitland import --dburl="$TL_TEST_SERVER_DATABASE_URL" --storage="$TL_TEST_STORAGE" 

# Current data
transitland fetch --dburl="$TL_TEST_SERVER_DATABASE_URL" --storage="$TL_TEST_STORAGE" --validation-report --validation-report-storage="$TL_TEST_STORAGE" --allow-local-fetch 
transitland import --dburl="$TL_TEST_SERVER_DATABASE_URL" --storage="$TL_TEST_STORAGE" --activate

# Sync again
transitland sync --dburl="$TL_TEST_SERVER_DATABASE_URL" "$SCRIPTDIR/server/server-test.dmfr.json"

# Supplemental data
psql -d "$TL_TEST_SERVER_DATABASE_URL" -f "$SCRIPTDIR/server/test_supplement.pgsql"

# Load census data
psql -d "$TL_TEST_SERVER_DATABASE_URL" -f "$SCRIPTDIR/server/census/census.pgsql"