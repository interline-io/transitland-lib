package extract

import (
	"testing"

	"github.com/interline-io/gotransit/gtcsv"
)

func TestExtract_Filter(t *testing.T) {
	em := NewMarker()
	reader, err := gtcsv.NewReader("../../testdata/bart.zip")
	if err != nil {
		t.Error(err)
	}
	fm := map[string][]string{}
	fm["trips.txt"] = []string{"3792107WKDY"}
	em.Filter(reader, fm)
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
