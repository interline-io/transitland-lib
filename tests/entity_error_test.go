package tests

import (
	"fmt"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tt"
)

func TestEntityErrors(t *testing.T) {
	reader, err := tlcsv.NewReader(testpath.RelPath("testdata/bad-entities"))
	if err != nil {
		t.Error(err)
	}
	if err := reader.Open(); err != nil {
		t.Error(err)
	}
	testutil.AllEntities(reader, func(ent tt.Entity) {
		t.Run(fmt.Sprintf("%s:%s", ent.Filename(), ent.EntityID()), func(t *testing.T) {
			errs := tt.CheckErrors(ent)
			expecterrs := testutil.GetExpectErrors(ent)
			testutil.CheckErrors(expecterrs, errs, t)
		})
	})
	if err := reader.Close(); err != nil {
		t.Error(err)
	}
}
