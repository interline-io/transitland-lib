#!/bin/sh
# Set up a new postgres database and import Natural Earth data.
set -ex
SCRIPTDIR=$(dirname "$0")
TL_TEST_STORAGE=$(dirname "$0")/server/tmp
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
    # Remove scheme (postgres:// or postgresql://)
    local url_without_scheme=$(echo "$url" | awk -F'://' '{print $2}')
    
    if echo "$url_without_scheme" | grep -q "@"; then
        # URL has user:pass@ - extract all components
        # user:pass@host:port/dbname
        local auth_part=$(echo "$url_without_scheme" | awk -F'@' '{print $1}')
        local host_part=$(echo "$url_without_scheme" | awk -F'@' '{print $2}')
        
        export PGUSER=$(echo "$auth_part" | awk -F':' '{print $1}')
        export PGPASSWORD=$(echo "$auth_part" | awk -F':' '{print $2}')
        export PGHOST=$(echo "$host_part" | awk -F'[:./]' '{print $1}')
        export PGPORT=$(echo "$host_part" | awk -F'[:./]' '{print $2}')
        export PGDATABASE=$(echo "$host_part" | awk -F'[/]' '{print $2}' | awk -F'[?]' '{print $1}')
    else
        # URL has no auth - extract host:port:dbname
        # host:port/dbname
        export PGHOST=$(echo "$url_without_scheme" | awk -F'[:./]' '{print $1}')
        export PGPORT=$(echo "$url_without_scheme" | awk -F'[:./]' '{print $2}')
        export PGDATABASE=$(echo "$url_without_scheme" | awk -F'[/]' '{print $2}' | awk -F'[?]' '{print $1}')
    fi
    echo "PGHOST=$PGHOST PGPORT=$PGPORT PGDATABASE=$PGDATABASE PGUSER=$PGUSER"
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