DB="$1"
OUTFILE="$2"
go build . && (cd cmd/gotransit && go build .)

echo $i
mkdir -p log
time GTFS_LOGLEVEL=debug cmd/gotransit/gotransit \
    copy \
    "postgres://localhost/${DB}?binary_parameters=yes&sslmode=disable" $OUTFILE 
