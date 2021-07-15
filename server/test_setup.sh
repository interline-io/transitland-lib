#!/bin/bash
transitland dmfr sync -dburl=$TL_TEST_SERVER_DATABASE_URL ../test/data/internal/server/server-test.dmfr.json
transitland dmfr fetch -dburl=$TL_TEST_SERVER_DATABASE_URL -gtfsdir=$TL_TEST_GTFSDIR
transitland dmfr import -dburl=$TL_TEST_SERVER_DATABASE_URL -gtfsdir=$TL_TEST_GTFSDIR -activate
psql $TL_TEST_SERVER_DATABASE_URL -c "refresh materialized view tl_mv_active_agency_operators;"