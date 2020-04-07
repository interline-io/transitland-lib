#!/bin/bash
set -ex
tmpdir=$(mktemp -d)
(cd cmd/gotransit/ && go install .) 
gotransit dmfr sync  testdata/dmfr/bayarea-tl.dmfr.json
gotransit dmfr fetch  -gtfsdir=${tmpdir}

echo "----- import -----"
gotransit dmfr import -gtfsdir=${tmpdir} --dryrun --fvid=1
gotransit dmfr import -gtfsdir=${tmpdir} --dryrun BA
gotransit dmfr import -gtfsdir=${tmpdir} --dryrun --fetched-since=2020-01-01
gotransit dmfr import -gtfsdir=${tmpdir} --dryrun --fetched-since=2050-01-01