package multireader

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/stretchr/testify/assert"
)

func TestMultireader(t *testing.T) {
	reader1, err := tlcsv.NewReader(testpath.RelPath("testdata/external/bart.zip"))
	if err != nil {
		t.Fatal(err)
	}
	reader2, err := tlcsv.NewReader(testpath.RelPath("testdata/external/caltrain.zip"))
	if err != nil {
		t.Fatal(err)
	}
	reader := NewReader(reader1, reader2)
	agencyIds := map[string]int{}
	for ent := range reader.Agencies() {
		agencyIds[ent.AgencyID] += 1
	}
	assert.Equal(t, 1, agencyIds["BART"])
	assert.Equal(t, 1, agencyIds["caltrain-ca-us"])
}
