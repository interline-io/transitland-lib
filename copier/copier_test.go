package copier

import (
	"fmt"
	"testing"

	"github.com/interline-io/transitland-lib/adapters/direct"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/stretchr/testify/assert"
)

type testCopierExpand struct{}

func (ext *testCopierExpand) Expand(ent tl.Entity, emap *tl.EntityMap) ([]tl.Entity, bool, error) {
	var ret []tl.Entity
	v, ok := ent.(*tl.Agency)
	if !ok {
		return nil, false, nil
	}
	for i := 0; i < 4; i++ {
		c := *v
		c.AgencyID = fmt.Sprintf("%s:%d", v.AgencyID, i)
		ret = append(ret, &c)
	}
	return ret, true, nil
}

func TestCopier_Expand(t *testing.T) {
	reader := direct.NewReader()
	reader.AgencyList = append(reader.AgencyList, tl.Agency{
		AgencyID:       "test",
		AgencyName:     "ok",
		AgencyPhone:    "555-123-4567",
		AgencyEmail:    "test@example.com",
		AgencyURL:      "http://example.com",
		AgencyTimezone: "America/Los_Angeles",
	})
	writer := direct.NewWriter()
	cp, err := NewCopier(reader, writer, Options{})
	if err := cp.AddExtension(&testCopierExpand{}); err != nil {
		t.Fatal(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	result := cp.Copy()
	if result.WriteError != nil {
		t.Fatal(err)
	}
	//
	agencyIds := map[string]int{}
	wreader, _ := writer.NewReader()
	for ent := range wreader.Agencies() {
		agencyIds[ent.AgencyID] += 1
	}
	assert.Equal(t, 4, len(agencyIds))
	assert.Equal(t, 1, agencyIds["test:0"])
	assert.Equal(t, 1, agencyIds["test:1"])
	assert.Equal(t, 1, agencyIds["test:2"])
	assert.Equal(t, 1, agencyIds["test:3"])
}
