
## Usage as a library

### Key library components

- Entity: An `Entity` is entity as specified by GTFS, such as an Agency, Route, Stop, etc.
- Reader: A `Reader` provides streams of GTFS entities over channels. The `tlcsv` and `tldb` modules provide CSV and PostgreSQL/SQLite support, respectively.
- Writer: A `Writer` accepts GTFS entities. As above, `tlcsv` and `tldb` provide basic implementations. Custom writers can also be used to support non-GTFS outputs, such as building a routing graph.
- Copier: A `Copier` reads a stream of GTFS entities from a `Reader`, checks each entity against a `Marker`, performs validation, applies any specified `Filters`, and sends to a `Writer`.
- Marker: A `Marker` selects which GTFS entities will be processed by a `Copier`. For example, selecting only entities related to a single trip or route.
- Filter: A `Filter` applies transformations to GTFS entities, such as converting extended route types to basic values, or modifying entity identifiers.
- Extension: An `Extension` provides support for additional types of GTFS entities.

See [godoc.org](https://godoc.org/github.com/interline-io/transitland-lib/tl) for package documentation.

### Install as a library

```bash
go get github.com/interline-io/transitland-lib
```

### Example of how to use as a library

A simple example of reading and writing GTFS entities from CSV ([full example](https://github.com/interline-io/transitland-lib/raw/master/internal/testreadme/main_test.go)):

```go
package main

import (
	"fmt"
	"testing"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
)

var URL = "https://github.com/interline-io/transitland-lib/raw/master/testdata/external/bart.zip"

func TestExample1(t *testing.T) {
	// Read stops from a GTFS url
	reader, _ := tlcsv.NewReader(URL)
	reader.Open()
	defer reader.Close()
	// Write to to the current directory
	writer, _ := tlcsv.NewWriter(".")
	writer.Open()
	// Copy stops
	for stop := range reader.Stops() {
		fmt.Println("Read Stop:", stop.StopID)
		eid, _ := writer.AddEntity(&stop)
		fmt.Println("Wrote stop:", eid)
	}
}
```

Database support is handled similary:

```go
func getReader() adapters.Reader {
	reader, _ := tlcsv.NewReader(URL)
	return reader
}

func TestExample2(t *testing.T) {
	reader := getReader()
	// Create a SQLite writer, in memory
	dburl := "sqlite3://:memory:"
	dbwriter, err := tldb.NewWriter(dburl)
	if err != nil {
		t.Fatalf("no reader available")
	}
	if err := dbwriter.Open(); err != nil {
		t.Fatalf("could not open database for writing")
	}
	if err := dbwriter.Create(); err != nil {
		t.Fatalf("could not find or create database schema")
	}
	for stop := range reader.Stops() {
		// A database writer AddEntity returns the primary key as a string.
		fmt.Println("Read Stop:", stop.StopID)
		eid, err := dbwriter.AddEntity(&stop)
		if err != nil {
			t.Fatalf("could not write entity to database")
		}
		fmt.Println("wrote stop to database:", eid)
	}
	// Read back from this source.
	dbreader, err := dbwriter.NewReader()
	if err != nil {
		t.Fatalf("could not get a new reader")
	}
	count := 0
	for stop := range dbreader.Stops() {
		fmt.Println("read stop from database:", stop.StopID)
		count++
	}
	if count != 50 {
		t.Errorf("got %d stops, expected 50", count)
	}
}
```

More advanced filtering operations can be performed using a `Copier`, which provides additional hooks for filtering, transformation, and validation:

```go
func TestExample3(t *testing.T) {
	reader := getReader()
	// Create a zip writer
	writer, err := tlcsv.NewWriter("filtered.zip")
	if err != nil {
		t.Fatalf("no writer available")
	}
	// Create a copier to stream, filter, and validate entities
	cp := copier.NewCopier(reader, writer)
	result := cp.Copy()
	if result.WriteError != nil {
		t.Fatalf("fatal copy error")
	}
	for _, err := range result.Errors {
		fmt.Println("error:", err)
	}
	for fn, count := range result.EntityCount {
		fmt.Printf("copied %d entities from %s\n", count, fn)
	}
}
```

See API docs at https://godoc.org/github.com/interline-io/transitland-lib
