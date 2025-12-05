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

func TestResult_CheckErrorThreshold(t *testing.T) {
	testCases := []struct {
		name           string
		entityCount    map[string]int
		errorCount     map[string]int
		refErrorCount  map[string]int
		thresholds     map[string]float64
		expectExceeded bool
		expectFiles    []string // files that should exceed threshold
	}{
		{
			name:           "no errors",
			entityCount:    map[string]int{"stops.txt": 100},
			errorCount:     map[string]int{},
			refErrorCount:  map[string]int{},
			thresholds:     map[string]float64{"*": 10},
			expectExceeded: false,
		},
		{
			name:           "under threshold",
			entityCount:    map[string]int{"stops.txt": 95},
			errorCount:     map[string]int{"stops.txt": 5},
			refErrorCount:  map[string]int{},
			thresholds:     map[string]float64{"*": 10},
			expectExceeded: false, // 5/100 = 5%
		},
		{
			name:           "over threshold",
			entityCount:    map[string]int{"stops.txt": 80},
			errorCount:     map[string]int{"stops.txt": 20},
			refErrorCount:  map[string]int{},
			thresholds:     map[string]float64{"*": 10},
			expectExceeded: true, // 20/100 = 20%
			expectFiles:    []string{"stops.txt"},
		},
		{
			name:           "combined entity and reference errors",
			entityCount:    map[string]int{"trips.txt": 85},
			errorCount:     map[string]int{"trips.txt": 10},
			refErrorCount:  map[string]int{"trips.txt": 5},
			thresholds:     map[string]float64{"*": 10},
			expectExceeded: true, // 15/100 = 15%
			expectFiles:    []string{"trips.txt"},
		},
		{
			name:           "one file over one under with default threshold",
			entityCount:    map[string]int{"stops.txt": 95, "trips.txt": 80},
			errorCount:     map[string]int{"stops.txt": 5, "trips.txt": 20},
			refErrorCount:  map[string]int{},
			thresholds:     map[string]float64{"*": 10},
			expectExceeded: true,
			expectFiles:    []string{"trips.txt"},
		},
		{
			name:           "per-file threshold stricter",
			entityCount:    map[string]int{"stops.txt": 95, "trips.txt": 85},
			errorCount:     map[string]int{"stops.txt": 5, "trips.txt": 15},
			refErrorCount:  map[string]int{},
			thresholds:     map[string]float64{"*": 20, "stops.txt": 3}, // stops.txt has stricter threshold
			expectExceeded: true,
			expectFiles:    []string{"stops.txt"}, // 5% > 3%
		},
		{
			name:           "per-file threshold more lenient",
			entityCount:    map[string]int{"stops.txt": 80, "trips.txt": 85},
			errorCount:     map[string]int{"stops.txt": 20, "trips.txt": 15},
			refErrorCount:  map[string]int{},
			thresholds:     map[string]float64{"*": 10, "stops.txt": 25}, // stops.txt has more lenient threshold
			expectExceeded: true,
			expectFiles:    []string{"trips.txt"}, // trips.txt uses default 10%, 15% > 10%
		},
		{
			name:           "empty thresholds",
			entityCount:    map[string]int{"stops.txt": 50},
			errorCount:     map[string]int{"stops.txt": 50},
			refErrorCount:  map[string]int{},
			thresholds:     nil,
			expectExceeded: false, // disabled
		},
		{
			name:           "exactly at threshold",
			entityCount:    map[string]int{"stops.txt": 90},
			errorCount:     map[string]int{"stops.txt": 10},
			refErrorCount:  map[string]int{},
			thresholds:     map[string]float64{"*": 10},
			expectExceeded: false, // 10% is not > 10%
		},
		{
			name:           "file-specific only no default",
			entityCount:    map[string]int{"stops.txt": 80, "trips.txt": 80},
			errorCount:     map[string]int{"stops.txt": 20, "trips.txt": 20},
			refErrorCount:  map[string]int{},
			thresholds:     map[string]float64{"stops.txt": 10}, // only stops.txt has threshold
			expectExceeded: true,
			expectFiles:    []string{"stops.txt"}, // trips.txt has no threshold so not checked
		},
		{
			name:           "zero threshold with errors",
			entityCount:    map[string]int{"stops.txt": 100},
			errorCount:     map[string]int{"stops.txt": 1},
			refErrorCount:  map[string]int{},
			thresholds:     map[string]float64{"*": 0}, // any error is failure
			expectExceeded: true,
			expectFiles:    []string{"stops.txt"},
		},
		{
			name:           "zero threshold with no errors",
			entityCount:    map[string]int{"stops.txt": 100},
			errorCount:     map[string]int{},
			refErrorCount:  map[string]int{},
			thresholds:     map[string]float64{"*": 0}, // any error is failure
			expectExceeded: false,
		},
		{
			name:           "zero threshold for specific file",
			entityCount:    map[string]int{"stops.txt": 100, "trips.txt": 100},
			errorCount:     map[string]int{"stops.txt": 0, "trips.txt": 1},
			refErrorCount:  map[string]int{},
			thresholds:     map[string]float64{"stops.txt": 0, "trips.txt": 10}, // stops.txt has zero tolerance
			expectExceeded: false,                                               // stops has 0 errors, trips has 1% < 10%
		},
		{
			name:           "zero threshold for specific file with error",
			entityCount:    map[string]int{"stops.txt": 100, "trips.txt": 100},
			errorCount:     map[string]int{"stops.txt": 1, "trips.txt": 5},
			refErrorCount:  map[string]int{},
			thresholds:     map[string]float64{"stops.txt": 0, "trips.txt": 10}, // stops.txt has zero tolerance
			expectExceeded: true,
			expectFiles:    []string{"stops.txt"}, // stops has 1 error with 0 threshold
		},
		{
			name:           "multiple files over threshold",
			entityCount:    map[string]int{"stops.txt": 80, "trips.txt": 80, "routes.txt": 80},
			errorCount:     map[string]int{"stops.txt": 20, "trips.txt": 20, "routes.txt": 5},
			refErrorCount:  map[string]int{},
			thresholds:     map[string]float64{"*": 10},
			expectExceeded: true,
			expectFiles:    []string{"stops.txt", "trips.txt"}, // both exceed, routes.txt is under
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NewResult(10)
			result.EntityCount = tc.entityCount
			result.SkipEntityErrorCount = tc.errorCount
			result.SkipEntityReferenceCount = tc.refErrorCount

			thresholdResult := result.CheckErrorThreshold(tc.thresholds)

			assert.Equal(t, tc.expectExceeded, thresholdResult.Exceeded, "Exceeded mismatch")

			if tc.expectExceeded {
				for _, fn := range tc.expectFiles {
					detail, ok := thresholdResult.Details[fn]
					assert.True(t, ok, "Expected file %s in details", fn)
					assert.True(t, detail.Exceeded, "Expected file %s to exceed threshold", fn)
				}
			}
		})
	}
}
