package main

import (
	"testing"

	tl "github.com/interline-io/transitland-lib"
)

func Test_exampleDB(t *testing.T) {
	// Just ensure this runs without panicing
	reader, err := tl.NewReader("../../testdata/external/bart.zip")
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
	reader, err := tl.NewReader("../../testdata/external/bart.zip")
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Open(); err != nil {
		t.Fatal(err)
	}
	exampleCopier(reader)
}
