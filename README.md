# Interline gotransit <!-- omit in toc -->

gotransit is a library and command-line tool for reading, writing, and processing transit data in [GTFS](http://gtfs.org) and related formats. The library is structured as a set of data sources, filters, and transformations that can be mixed together in a variety of ways to create processing pipelines. The library supports the [DMFR](https://github.com/transitland/distributed-mobility-feed-registry) format to specify multiple input feeds.

## Table of Contents <!-- omit in toc -->
<!-- to update use https://marketplace.visualstudio.com/items?itemName=yzhang.markdown-all-in-one -->
- [Installation](#installation)
	- [Download prebuilt binary](#download-prebuilt-binary)
	- [Install on MacOS using Homebrew](#install-on-macos-using-homebrew)
	- [To build from source](#to-build-from-source)
	- [Installing with SQLite Support](#installing-with-sqlite-support)
- [Usage as a CLI tool](#usage-as-a-cli-tool)
	- [`validate` command](#validate-command)
	- [`copy` command](#copy-command)
	- [`extract` command](#extract-command)
	- [`dmfr` command](#dmfr-command)
- [Usage as a library](#usage-as-a-library)
	- [Key library components](#key-library-components)
	- [Example of how to use as a library](#example-of-how-to-use-as-a-library)
- [Included Readers and Writers](#included-readers-and-writers)
- [Development](#development)
	- [Releases](#releases)
- [Licenses](#licenses)

## Installation

### Download prebuilt binary

Linux and macOS binaries are attached to each [release](https://github.com/interline-io/gotransit/releases).

### Install on MacOS using Homebrew

To install using the [Gotransit formula for Homebrew](https://github.com/interline-io/homebrew-gotransit):

```sh
brew install interline-io/gotransit/gotransit
```

### To build from source

```bash
go get github.com/interline-io/gotransit
```

Main dependencies:
- `twpayne/go-geom`
- `jmoiron/sqlx`
- `Masterminds/squirrel`
- `lib/pq`
- `mattn/go-sqlite3` (see below)

### Installing with SQLite Support

SQLite CGO support, and is not included in the static release builds. To enable support, compile locally with `CGO_ENABLED=1`.

## Usage as a CLI tool

The main subcommands are:
- [validate](#validate-command)
- [copy](#copy-command)
- [extract](#extract-command)
- [dmfr](#dmfr-command)

### `validate` command

The validate command performs a basic validation on a data source and writes the results to standard out.

```
$ gotransit validate --help
Usage: validate <reader>
  -ext value
    	Include GTFS Extension
```

Example: 

```sh
$ gotransit validate "http://www.caltrain.com/Assets/GTFS/caltrain/CT-GTFS.zip"
```

### `copy` command

The copy command performs a basic copy from a reader to a writer. By default, any entity with errors will be skipped and not written to output. This can be ignored with `-allow-entity-errors` to ignore simple errors and `-allow-reference-errors` to ignore entity relationship errors, such as a reference to a non-existent stop.

```
$ gotransit copy --help
Usage: copy <reader> <writer>
  -allow-entity-errors
    	Allow entities with errors to be copied
  -allow-reference-errors
    	Allow entities with reference errors to be copied
  -create
    	Create a basic database schema if none exists
  -ext value
    	Include GTFS Extension
  -fvid int
    	Specify FeedVersionID when writing to a database
```

Example:

```sh
$ gotransit copy --allow-entity-errors "http://www.caltrain.com/Assets/GTFS/caltrain/CT-GTFS.zip" output.zip

$ unzip -p output.zip agency.txt
agency_id,agency_name,agency_url,agency_timezone,agency_lang,agency_phone,agency_fare_url,agency_email
1000,Caltrain,http://www.caltrain.com,America/Los_Angeles,en,800-660-4287,,
  ```

### `extract` command

The extract command extends the basic copy command with a number of additional options and transformations. It can be used to pull out a single route or trip, interpolate stop times, override a single value on an entity, etc. This is a separate command to keep the basic copy command simple while allowing the extract command to grow and add more features over time.

```
$ gotransit extract --help
Usage: extract <input> <output>
  -allow-entity-errors
    	Allow entities with errors to be copied
  -allow-reference-errors
    	Allow entities with reference errors to be copied
  -create
    	Create a basic database schema if none exists
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
  -fvid int
    	Specify FeedVersionID when writing to a database
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

### `dmfr` command

_under development_

The `dmfr` command enables processing multiple feeds at once using a catalog in the [Distributed Mobility Feed Registry]([dmfr](https://github.com/transitland/distributed-mobility-feed-registry)) format. It provides several additional subcommands for reading DMFR files, synchronizing these feeds to a database, downloading the latest versions of each feed, and automatically importing the feeds into a database. It provides the foundation for [Transitland v2](https://transit.land/news/2019/10/17/tlv2.html).

This command is still under active development and may change in future releases. Please see [DMFR Command help](dmfr-command.md).

## Usage as a library

### Key library components

- Entity: An `Entity` is entity as specified by GTFS, such as an Agency, Route, Stop, etc.
- Reader: A `Reader` provides streams of GTFS entities over channels. The `gtcsv` and `gtdb` modules provide CSV and Postgres/SQLite support, respectively.
- Writer: A `Writer` accepts GTFS entities. As above, `gtcsv` and `gtdb` provide basic implementations. Custom writers can also be used to support non-GTFS outputs, such as building a routing graph.
- Copier: A `Copier` reads a stream of GTFS entities from a `Reader`, checks each entity against a `Marker`, performs validation, applies any specified `Filters`, and sends to a `Writer`.
- Marker: A `Marker` selects which GTFS entities will be processed by a `Copier`. For example, selecting only entities related to a single trip or route.
- Filter: A `Filter` applies transformations to GTFS entities, such as converting extended route types to basic values, or modifying entity identifiers.
- Extension: An `Extension` provides support for additional types of GTFS entities.

### Example of how to use as a library

A simple example of reading and writing GTFS entities from CSV:

```go
import (
	"fmt"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/copier"
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
	for stop := range reader.Stops() {
		fmt.Println("Read Stop:", stop.StopID)
		eid, err := writer.AddEntity(&stop)
		check(err)
		fmt.Println("Wrote Stop:", eid)
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
	for stop := range reader.Stops() {
		// Preserves StopID but also assigns an integer ID (returned as string).
		fmt.Println("Read Stop:", stop.StopID)
		eid, err := dbwriter.AddEntity(&stop)
		check(err)
		fmt.Println("Wrote Stop:", eid)
	}
	// Read back from this source.
	dbreader, err := dbwriter.NewReader()
	check(err)
	for stop := range dbreader.Stops() {
		fmt.Println("Read Stop:", stop.StopID)
	}
	// Query database
}
```

More advanced operations can be performed using a `Copier`, which provides additional hooks for filtering, transformation, and validation:

```go
func exampleCopier(reader gotransit.Reader) {
	writer, err := gtcsv.NewWriter("filtered.zip")
	check(err)
	check(writer.Open())
	defer writer.Close()
	cp := copier.NewCopier(reader, writer)
	result := cp.Copy()
	for _, err := range result.Errors {
		fmt.Println("Error:", err)
	}
	for fn, count := range result.Count {
		fmt.Printf("Copied %d entities from %s\n", count, fn)
	}
}
```

See API docs at https://godoc.org/github.com/interline-io/gotransit

## Included Readers and Writers

| Target                   | Module  | Supports Read | Supports Write |
| ------------------------ | ------- | ------------- | -------------- |
| CSV                      | `gtcsv` | ✅             | ✅              |
| SQLite (with SQLite) | `gtdb`  | ✅             | ✅              |
| Postgres (with PostGIS)  | `gtdb`  | ✅             | ✅              |

We welcome the addition of more readers and writers.

## Development

gotransit follows Go coding conventions.

GitHub Actions runs all tests, stores code coverage reports as artifacts, and cuts releases using [GoReleaser](https://github.com/goreleaser/goreleaser).

### Releases

Releases follow [Semantic Versioning](https://semver.org/) conventions.

To cut a new release:

1. Tag the `master` branch with the next SemVer version (for example: `v0.2.0`).
2. GitHub Actions will run [GoReleaser](https://github.com/goreleaser/goreleaser) and create a GitHub release on this repository.

## Licenses

gotransit is released under a "dual license" model:

- open-source for use by all under the [GPLv3](LICENSE) license
- also available under a flexible commercial license from [Interline](mailto:info@interline.io)

