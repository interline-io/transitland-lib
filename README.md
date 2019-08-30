# Interline gotransit

gotransit is a library and command-line tool for reading, writing, and processing transit data in [GTFS](http://gtfs.org) and related formats. The library is structured as a set of data sources, filters, and transformations that can be mixed together in a variety of ways to create processing pipelines. The library supports the [DMFR](https://github.com/transitland/distributed-mobility-feed-registry) format to specify multiple input feeds.

## Key components

- Entity: An `Entity` is entity as specified by GTFS, such as an Agency, Route, Stop, etc.
- Reader: A `Reader` provides streams of GTFS entities over channels. The `gtcsv` and `gtdb` modules provide CSV and Postgres/SQLite support, respectively.
- Writer: A `Writer` accepts GTFS entities. As above, `gtcsv` and `gtdb` provide basic implementations. Custom writers can also be used to support non-GTFS outputs, such as building a routing graph.
- Copier: A `Copier` reads a stream of GTFS entities from a `Reader`, checks each entity against a `Marker`, performs validation, applies any specified `Filters`, and sends to a `Writer`.
- Marker: A `Marker` selects which GTFS entities will be processed by a `Copier`. For example, selecting only entities related to a single trip or route.
- Filter: A `Filter` applies transformations to GTFS entities, such as converting extended route types to basic values, or modifying entity identifiers.
- Extension: An `Extension` provides support for additional types of GTFS entities.


## Installation

```bash
go get github.com/interline-io/gotransit
```

Linux binaries are attached to each [release](https://github.com/interline-io/gotransit/releases).

Main dependencies (handled by `go.mod`):

- `twpayne/go-geom`
- `lib/pq`
- `jmoiron/sqlx`
- `Masterminds/squirrel`
- `mattn/go-sqlite3` (see below)

SQLite / SpatiaLite requires CGO support and an available `gcc` compiler, as well as the [SpatiaLite](https://www.gaia-gis.it/fossil/libspatialite/index) shared library installed. You can generally install this using your system's package manager, e.g. `apt-get install libsqlite3-mod-spatialite` or `brew install libspatialite`. This will be an optional dependency with a build flag in future releases.

## Usage as a CLI tool

### validate command

The validate command performs a basic validation on a data source and writes the results to standard out.

```
$ gotransit validate --help
Usage of validate:
  -ext value
    	Include GTFS Extension
```

TODO: AN EXAMPLE WITH A REAL VALIDATION ERROR AND FIX PRINTING

```sh
$ gotransit validate "http://www.caltrain.com/Assets/GTFS/caltrain/CT-GTFS.zip"
```

### copy command

The copy command performs a basic copy from a reader to a writer. By default, any entity with errors will be skipped and not written to output. This can be ignored with `-allow-entity-errors` to ignore simple errors and `-allow-reference-errors` to ignore entity relationship errors, such as a reference to a non-existent stop.

```
$ gotransit copy --help
Usage of copy:
  -allow-entity-errors
    	Allow entity-level errors
  -allow-reference-errors
    	Allow reference errors
  -create
    	Create
  -ext value
    	Include GTFS Extension
```

TODO: SET FILE CREATION TIMES IN ZIP, USE EXAMPLE WITH AN ERROR

Example:

```sh
$ gotransit copy --allow-entity-errors "http://www.caltrain.com/Assets/GTFS/caltrain/CT-GTFS.zip" output.zip

$ unzip -p ../output.zip agency.txt
agency_id,agency_name,agency_url,agency_timezone,agency_lang,agency_phone,agency_fare_url,agency_email
1000,Caltrain,http://www.caltrain.com,America/Los_Angeles,en,800-660-4287,,
  ```

### extract command

The extract command extends the basic copy command with a number of additional options and transformations.

```
$ gotransit extract --help
Usage of extract:
  -allow-entity-errors
    	Allow entity-level errors
  -allow-reference-errors
    	Allow reference errors
  -create
    	Create
  -create-missing-shapes
    	Create missing Shapes from Trip stop-to-stop geometries
  -ext value
    	Include GTFS Extension
  -extract-agency value
    	Extract Agency
  -extract-calendar value
    	Extract Calendar
  -extract-route value
    	Extract Route
  -extract-route-type value
    	Extract Routes matching route_type
  -extract-stop value
    	Extract Stop
  -extract-trip value
    	Extract Trip
  -interpolate-stop-times
    	Interpolate missing StopTime arrival/departure values
  -normalize-service-ids
    	Create Calendar entities for CalendarDate service_id's
  -set value
    	Set values on output; format is filename,id,key,value
  -use-basic-route-types
    	Collapse extended route_type's into basic GTFS values
```

Example:

```sh
# Extract a single trip from the Caltrain GTFS, and rename the agency to "caltrain".
$ gotransit extract -extract-trip 305 -set agency.txt,1000,agency_id,caltrain "http://www.caltrain.com/Assets/GTFS/caltrain/CT-GTFS.zip" output2.zip

# Note renamed agency
$ unzip -p output2.zip agency.txt
agency_id,agency_name,agency_url,agency_timezone,agency_lang,agency_phone,agency_fare_url,agency_email
caltrain,Caltrain,http://www.caltrain.com,America/Los_Angeles,en,800-660-4287,,

# Only entities related to the specified trip are included in the output.
$ unzip -p output2.zip trips.txt
route_id,service_id,trip_id,trip_headsign,trip_short_name,direction_id,block_id,shape_id,wheelchair_accessible,bikes_allowed
12867,c_16869_b_19500_d_31,305,San Francisco Caltrain Station,305,0,,p_692594,0,0

$ unzip -p output2.zip routes.txt
route_id,agency_id,route_short_name,route_long_name,route_desc,route_type,route_url,route_color,route_text_color,route_sort_order
12867,caltrain,Bullet,Baby Bullet,,2,,E31837,ffffff,2

$ unzip -p output2.zip stop_times.txt
trip_id,arrival_time,departure_time,stop_id,stop_sequence,stop_headsign,pickup_type,drop_off_type,shape_dist_traveled,timepoint
305,05:45:00,05:45:00,70261,1,,0,0,0.00000,1
305,06:01:00,06:01:00,70211,2,,0,0,17498.98397,1
305,06:09:00,06:09:00,70171,3,,0,0,27096.41601,1
305,06:19:00,06:19:00,70111,4,,0,0,42877.37732,1
305,06:28:00,06:28:00,70061,5,,0,0,53641.84115,1
305,06:47:00,06:47:00,70011,6,,0,0,75372.02742,1
```

## Usage as a library

A simple example of reading and writing GTFS entities from CSV:

```go
package main

import (
    "github.com/interline-io/gotransit"
    "github.com/interline-io/gotransit/gtcsv"
    "github.com/interline-io/gotransit/gtdb"
)

func main() {
    // Saves to a temporary file, removed upon Close().
    // Local paths to zip files and plain directories are also supported.
    url := "http://www.caltrain.com/Assets/GTFS/caltrain/CT-GTFS.zip"
    reader, err := gtcsv.NewReader(url)
    check(err)
    check(reader.Open())
    defer reader.Close()

    // Create a CSV writer
    // Writes to temporary directory, creates zip upon Close().
    writer, err := gtcsv.NewWriter("output.zip") 
    check(err)
    check(writer.Open())
    // Copy from Reader to Writer.
    for agency := range reader.Stops() {
        fmt.Println("Read Agency:", agency.AgencyID)
        eid, err := writer.AddEntity(&agency)
        check(err)
        fmt.Println("Wrote Agency:", eid)
    }
    // Go ahead and close, check for errors
    check(writer.Close())
}

func check(err error) {
    if err != nil {
        panic(err)
    }
}
```

Database support is handled similary:

```go
func exampleDB(reader gotransit.Reader) {
     // Create a SQLite writer, in memory
    dburl := "sqlite3://:memory:"
    dbwriter, err := gtdb.NewWriter(dburl)
    check(err)
    check(dbwriter.Open())
    check(dbwriter.Create()) // Install schema.
    for agency := range reader.Agencies() {
        // Preserves AgencyID but also assigns an integer ID (returned as string).
        fmt.Println("Read Agency:", agency.AgencyID)
        eid, err := dbwriter.AddEntity(&agency)
        check(err)
        fmt.Println("Wrote Agency:", eid)
    }
    // Read back from this source.
    dbreader := dbwriter.NewReader()
    for agency := range dbreader.Agencies() {
        fmt.Println("Read Agency:", agency.AgencyID)
    }
    // Query database
}
```

More advanced operations can be performed using a `Copier`, which provides additional hooks for filtering, transformation, and validation:

```go
func exampleCopier(reader gotransit.Reader) {
    writer, err := gtcsv.NewWriter("filtered.zip")
    check(err)
    check(writer.Open)
    defer writer.Close()
    cp := copier.NewCopier(reader, &writer)
    result := cp.Copy()
    for _, err := range result.Errors {
        fmt.Println("Error:", err)
    }
    for fn,count := range result.Count {
        fmt.Printf("Copied %d entities from %s\n", count, fn)
    }
}
```

See API docs at https://godoc.org/github.com/interline-io/gotransit

## Development

gotransit follows Go coding conventions.

CircleCI runs all tests and stores code coverage reports as artifacts at https://circleci.com/gh/interline-io/gotransit

### Releases

Releases follow [Semantic Versioning](https://semver.org/) conventions.

To cut a new release:

1. Tag the `master` branch with the next SemVer version (for example: `v0.2.0`).
2. CircleCI will run [GoReleaser](https://github.com/goreleaser/goreleaser) and create a GitHub release on this repository.

## Licenses

GoTransit is released under a "dual license" model:

- open-source for use by all under the GPLv3 license
- also available under a flexible commercial license from Interline