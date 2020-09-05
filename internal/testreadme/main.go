package main

import (
	"fmt"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tldb"
)

func main() {
	// Saves to a temporary file, removed upon Close().
	// Local paths to zip files and plain directories are also supported.
	url := "http://www.caltrain.com/Assets/GTFS/caltrain/CT-GTFS.zip"
	reader, err := tlcsv.NewReader(url)
	check(err)
	check(reader.Open())
	defer reader.Close()
	// Create a CSV writer
	// Writes to temporary directory, creates zip upon Close().
	writer, err := tlcsv.NewWriter("output.zip")
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

func exampleDB(reader tl.Reader) {
	// Create a SQLite writer, in memory
	dburl := "sqlite3://:memory:"
	dbwriter, err := tldb.NewWriter(dburl)
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

func exampleCopier(reader tl.Reader) {
	writer, err := tlcsv.NewWriter("/tmp/filtered.zip")
	check(err)
	check(writer.Open())
	defer writer.Close()
	cp := copier.NewCopier(reader, writer)
	result := cp.Copy()
	for _, err := range result.Errors {
		fmt.Println("Error:", err)
	}
	for fn, count := range result.EntityCount {
		fmt.Printf("Copied %d entities from %s\n", count, fn)
	}
}
