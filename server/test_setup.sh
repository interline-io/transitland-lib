#!/bin/bash
transitland dmfr sync ../test/data/server/server-test.dmfr.json
transitland dmfr fetch
transitland dmfr import -activate
psql -c "refresh materialized view tl_mv_active_agency_operators;"