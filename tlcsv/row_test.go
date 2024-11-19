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

	for ent, err := range adapters.ReadEntitiesIter[gtfs.Stop](adapter) {
		fmt.Println("ent:", ent.StopName, "err:", err)
	}
}
