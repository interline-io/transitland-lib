DB="$1"
FILES="${@:2}"
go build . && (cd cmd/gotransit && go build .)
dropdb --if-exists ${DB} && createdb ${DB}

mkdir -p logs
rm -rf logs/*.log
for i in ${FILES}; do 
    echo $i
    mkdir -p log
    time GTFS_LOGLEVEL=trace cmd/gotransit/gotransit \
        copy \
        --create \
        --newfv \
        --normalizeserviceids \
        --createshapes \
        --visited \
        --interpolate \
        $i "postgres://localhost/${DB}?binary_parameters=yes&sslmode=disable" | tee logs/`basename $i`.log
done
