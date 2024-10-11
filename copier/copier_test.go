package copier

import (
	"fmt"
	"testing"

	"github.com/interline-io/transitland-lib/adapters/direct"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tt"
	"github.com/stretchr/testify/assert"
)

type testCopierExpand struct{}

func (ext *testCopierExpand) Expand(ent tl.Entity, emap *tt.EntityMap) ([]tl.Entity, bool, error) {
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

////////

// TODO: figure out why the fast benchmark is fast and the slow benchmark is slow
// This relates to copier.checkBatch: why is it faster when
// checkEntity is BEFORE appending to the batch slice,
// vs. appending always and then calling checkEntity during
// other filtering (as in CopyEntity)
var wtfBatchSize = 1_000_000

func BenchmarkWtfSlow(b *testing.B) {
	testWtfCopyEntities := func(ents []tl.Entity) {
		okEnts := make([]tl.Entity, 0, len(ents))
		for _, ent := range ents {
			if err := testWtfCheck(ent); err != nil {
				okEnts = append(okEnts, ent)
			}
		}
		testWtfWriteEntities(okEnts)
	}
	testWtfCheckBatch := func(ents []tl.Entity, ent tl.Entity) []tl.Entity {
		if len(ents) >= wtfBatchSize || ent == nil {
			testWtfCopyEntities(ents)
			return nil
		}
		ents = append(ents, ent)
		return ents
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		var ents []tl.Entity
		for i := 0; i < wtfBatchSize; i++ {
			ents = testWtfCheckBatch(ents, &tl.StopTime{})
		}
		testWtfCheckBatch(ents, nil)
	}
}

func BenchmarkWtfFast(b *testing.B) {
	testWtfCopyEntities := func(ents []tl.Entity) {
		testWtfWriteEntities(ents)
	}
	testWtfCheckBatch := func(ents []tl.Entity, ent tl.Entity) []tl.Entity {
		if len(ents) >= wtfBatchSize || ent == nil {
			testWtfCopyEntities(ents)
			return nil
		}
		if err := testWtfCheck(ent); err == nil {
			ents = append(ents, ent)
		}
		return ents
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		var ents []tl.Entity
		for i := 0; i < wtfBatchSize; i++ {
			ents = testWtfCheckBatch(ents, &tl.StopTime{})
		}
		testWtfCheckBatch(ents, nil)
	}
}

func testWtfCheck(ent tl.Entity) error {
	a := ent.Filename()
	b := ent.EntityID()
	_ = a
	_ = b
	return nil
}

func testWtfWriteEntities(ents []tl.Entity) {
	b := len(ents)
	_ = b
	_ = ents
}
