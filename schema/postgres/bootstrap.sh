#!/bin/sh
# Set up a new postgres database and import Natural Earth data.
# Optionally import GTFS feeds from directories or DMFR files.
# Usage: bootstrap.sh [directory|dmfr-file]...
# Environment variables:
#   TL_STORAGE - Storage path for fetched feeds (default: tmp)
#   WORKERS - Number of worker threads (default: 1)
set -e
SCRIPTDIR=$(dirname "$0")
WORKERS="${WORKERS:-1}"
TL_STORAGE="${TL_STORAGE:-tmp}"

# Wait for database to accept connections
${SCRIPTDIR}/wait-for-it.sh "${PGHOST}:${PGPORT}"

# Fail if db already exists -- this is a bootstrap script after all!
createdb "${PGDATABASE}"

# Database schema
transitland dbmigrate up

# Load Natural Earth - ogr2ogr is required for this.
transitland dbmigrate natural-earth

# Import GTFS feeds from directories or DMFR files
for arg in "$@"; do
    if [ -d "$arg" ]; then
        echo "Importing GTFS feeds from directory: $arg"
        mkdir -p "$TL_STORAGE"
        tmp_dmfr="$(mktemp)"
        transitland dmfr from-dir "$arg" > "$tmp_dmfr"
        transitland sync "$tmp_dmfr"
        transitland fetch --dmfr "$tmp_dmfr" --allow-local-fetch --storage="$TL_STORAGE" --workers="$WORKERS"
        transitland import --dmfr "$tmp_dmfr" --latest --workers="$WORKERS"
        rm -f "$tmp_dmfr"
    elif [ -f "$arg" ]; then
        echo "Importing GTFS feeds from DMFR: $arg"
        mkdir -p "$TL_STORAGE"
        transitland sync "$arg"
        transitland fetch --dmfr "$arg" --storage="$TL_STORAGE" --workers="$WORKERS"
        transitland import --dmfr "$arg" --latest --workers="$WORKERS"
    else
        echo "Warning: $arg is not a directory or file, skipping"
    fi
done
