package extract

import (
	"fmt"
	"testing"

	"github.com/interline-io/gotransit/gtcsv"
)

func printVisited(em *extractMarker) {
	for k, v := range em.found {
		fmt.Println(*k, v)
	}
}

func TestExtract_VisitAndMark(t *testing.T) {
	em := NewExtractMarker()
	reader, err := gtcsv.NewReader("../testdata/bart.zip")
	if err != nil {
		t.Error(err)
	}
	em.VisitAndMark(reader)
}

func TestExtract_Filter(t *testing.T) {
	em := NewExtractMarker()
	reader, err := gtcsv.NewReader("../testdata/bart.zip")
	if err != nil {
		t.Error(err)
	}
	em.VisitAndMark(reader)
	fm := map[string][]string{}
	fm["trips.txt"] = []string{"3792107WKDY"}
	em.Filter(fm)
	if !em.IsMarked("stops.txt", "MCAR") {
		t.Error("expected stop MCAR")
	}
	if em.IsMarked("stops.txt", "FTVL") {
		t.Error("expected no stop FTVL")
	}
	if !em.IsMarked("agency.txt", "BART") {
		t.Error("expected agency BART")
	}
	if em.IsMarked("routes.txt", "03") {
		t.Error("expected no route 03")
	}
}
