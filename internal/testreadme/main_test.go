package main

import (
	"fmt"
	"testing"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
	_ "github.com/interline-io/transitland-lib/tldb/tlsqlite"
)

// var URL = "https://github.com/interline-io/transitland-lib/raw/master/testdata/external/bart.zip"
var URL = "../../testdata/external/bart.zip"

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

func TestExample3(t *testing.T) {
	reader := getReader()
	// Create a zip writer
	writer, err := tlcsv.NewWriter("filtered.zip")
	if err != nil {
		t.Fatalf("no writer available")
	}
	// Create a copier to stream, filter, and validate entities
	cp, err := copier.NewCopier(reader, writer, copier.Options{})
	if err != nil {
		t.Fatal(err)
	}
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
