package direct

import (
	"testing"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/stretchr/testify/assert"
)

func TestWriter(t *testing.T) {
	writer := NewWriter()
	writer.AddEntity(&tl.Agency{AgencyID: "test", AgencyName: "ok"})
	reader, err := writer.NewReader()
	if err != nil {
		t.Fatal(err)
	}
	agencyIds := map[string]int{}
	for ent := range reader.Agencies() {
		agencyIds[ent.AgencyID] += 1
	}
	assert.Equal(t, 1, agencyIds["test"])
}
