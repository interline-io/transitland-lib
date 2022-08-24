package direct

import (
	"testing"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/stretchr/testify/assert"
)

func TestReader(t *testing.T) {
	reader := NewReader()
	reader.AgencyList = append(reader.AgencyList, tl.Agency{AgencyID: "test", AgencyName: "ok"})
	agencyIds := map[string]int{}
	for ent := range reader.Agencies() {
		agencyIds[ent.AgencyID] += 1
	}
	assert.Equal(t, 1, agencyIds["test"])
}
