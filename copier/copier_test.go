package copier

import (
	"context"
	"fmt"
	"testing"

	"github.com/interline-io/transitland-lib/adapters/direct"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

type testCopierExpand struct{}

func (ext *testCopierExpand) Expand(ent tt.Entity, emap *tt.EntityMap) ([]tt.Entity, bool, error) {
	var ret []tt.Entity
	v, ok := ent.(*gtfs.Agency)
	if !ok {
		return nil, false, nil
	}
	for i := 0; i < 4; i++ {
		c := *v
		c.AgencyID.Set(fmt.Sprintf("%s:%d", v.AgencyID.Val, i))
		ret = append(ret, &c)
	}
	return ret, true, nil
}

func TestCopier_Expand(t *testing.T) {
	reader := direct.NewReader()
	reader.AgencyList = append(reader.AgencyList, gtfs.Agency{
		AgencyID:       tt.NewString("test"),
		AgencyName:     tt.NewString("ok"),
		AgencyPhone:    tt.NewString("555-123-4567"),
		AgencyEmail:    tt.NewEmail("test@example.com"),
		AgencyURL:      tt.NewUrl("http://example.com"),
		AgencyTimezone: tt.NewTimezone("America/Los_Angeles"),
	})
	writer := direct.NewWriter()
	cpOpts := Options{}
	cpOpts.AddExtension(&testCopierExpand{})

	_, err := CopyWithOptions(context.Background(), reader, writer, cpOpts)
	if err != nil {
		t.Fatal(err)
	}
	//
	agencyIds := map[string]int{}
	wreader, _ := writer.NewReader()
	for ent := range wreader.Agencies() {
		agencyIds[ent.AgencyID.Val] += 1
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
	testWtfCopyEntities := func(ents []tt.Entity) {
		okEnts := make([]tt.Entity, 0, len(ents))
		for _, ent := range ents {
			if err := testWtfCheck(ent); err != nil {
				okEnts = append(okEnts, ent)
			}
		}
		testWtfWriteEntities(okEnts)
	}
	testWtfCheckBatch := func(ents []tt.Entity, ent tt.Entity) []tt.Entity {
		if len(ents) >= wtfBatchSize || ent == nil {
			testWtfCopyEntities(ents)
			return nil
		}
		ents = append(ents, ent)
		return ents
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		var ents []tt.Entity
		for i := 0; i < wtfBatchSize; i++ {
			ents = testWtfCheckBatch(ents, &gtfs.StopTime{})
		}
		testWtfCheckBatch(ents, nil)
	}
}

func BenchmarkWtfFast(b *testing.B) {
	testWtfCopyEntities := func(ents []tt.Entity) {
		testWtfWriteEntities(ents)
	}
	testWtfCheckBatch := func(ents []tt.Entity, ent tt.Entity) []tt.Entity {
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
		var ents []tt.Entity
		for i := 0; i < wtfBatchSize; i++ {
			ents = testWtfCheckBatch(ents, &gtfs.StopTime{})
		}
		testWtfCheckBatch(ents, nil)
	}
}

func testWtfCheck(ent tt.Entity) error {
	a := ent.Filename()
	b := ent.EntityID()
	_ = a
	_ = b
	return nil
}

func testWtfWriteEntities(ents []tt.Entity) {
	b := len(ents)
	_ = b
	_ = ents
}
