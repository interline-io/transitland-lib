#!/bin/sh
# Set up a new postgres database and import Natural Earth data.
set -e
SCRIPTDIR=$(dirname "$0")

# Wait for database to accept connections
${SCRIPTDIR}/wait-for-it.sh "${PGHOST}:${PGPORT}"

# Fail if db already exists -- this is a bootstrap script after all!
createdb "${PGDATABASE}"

# Database schema
transitland dbmigrate up

# Load Natural Earth - ogr2ogr is required for this.
transitland dbmigrate natural-earth
