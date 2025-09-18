package direct

import (
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

func TestReader(t *testing.T) {
	reader := NewReader()
	reader.AgencyList = append(reader.AgencyList, gtfs.Agency{AgencyID: tt.NewString("test"), AgencyName: tt.NewString("ok")})
	agencyIds := map[string]int{}
	for ent := range reader.Agencies() {
		agencyIds[ent.AgencyID.Val] += 1
	}
	assert.Equal(t, 1, agencyIds["test"])
}
