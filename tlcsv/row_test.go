package tlcsv

import (
	"fmt"
	"testing"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/internal/testpath"
)

func TestReadRowsIter(t *testing.T) {
	adapter := &ZipAdapter{path: testpath.RelPath("testdata/example.zip")}
	if err := adapter.Open(); err != nil {
		t.Error(err)
		return
	}

	it, errf := adapters.ReadEntitiesIter[gtfs.Stop](adapter)
	for ent := range it {
		fmt.Println("ent:", ent.StopName)
	}
	if err := errf(); err != nil {
		t.Error(err)
	}
}
