package empty

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReader(t *testing.T) {
	reader := NewReader()
	agencyIds := map[string]int{}
	for ent := range reader.Agencies() {
		agencyIds[ent.AgencyID] += 1
	}
	assert.Equal(t, 0, len(agencyIds))
}
