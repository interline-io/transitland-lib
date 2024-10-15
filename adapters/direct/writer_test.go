package direct

import (
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

func TestWriter(t *testing.T) {
	writer := NewWriter()
	writer.AddEntity(&gtfs.Agency{AgencyID: tt.NewString("test"), AgencyName: tt.NewString("ok")})
	reader, err := writer.NewReader()
	if err != nil {
		t.Fatal(err)
	}
	agencyIds := map[string]int{}
	for ent := range reader.Agencies() {
		agencyIds[ent.AgencyID.Val] += 1
	}
	assert.Equal(t, 1, agencyIds["test"])
}
