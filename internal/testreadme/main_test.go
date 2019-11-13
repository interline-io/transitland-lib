package main

import (
	"testing"

	"github.com/interline-io/gotransit"
)

func Test_exampleDB(t *testing.T) {
	// Just ensure this runs without panicing
	reader, err := gotransit.NewReader("../../testdata/external/bart.zip")
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Open(); err != nil {
		t.Fatal(err)
	}
	exampleDB(reader)
}

func Test_exampleCopier(t *testing.T) {
	// Just ensure this runs without panicing
	reader, err := gotransit.NewReader("../../testdata/external/bart.zip")
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Open(); err != nil {
		t.Fatal(err)
	}
	exampleCopier(reader)
}
