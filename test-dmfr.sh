#!/bin/bash
set -e
tmpdir=$(mktemp -d)
(cd cmd/gotransit/ && go install .) 
gotransit dmfr sync  testdata/dmfr/bayarea-tl.dmfr.json
gotransit dmfr fetch  -gtfsdir=${tmpdir}
gotransit dmfr import -gtfsdir=${tmpdir}