#!/bin/sh
psql -f activate_feed_version.pgsql
psql -f after_feed_version_import.pgsql
psql -f assign_agency_places.pgsql
psql -f calculate_route_headways.pgsql
psql -f services_on_date.pgsql
psql -f unimport_feed_version.pgsql
psql -f views.pgsql