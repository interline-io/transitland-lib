package builders

import (
	"testing"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tlcsv"
)

func TestBuilders(t *testing.T) {
	bartZip := testutil.ExampleFeedBART.URL
	// bartZip = "/Users/irees/src/interline-io/tlv2/test/data/bootstrap/f-9q9-actransit.zip"
	reader, err := tlcsv.NewReader(bartZip)
	if err != nil {
		panic(err)
	}
	if err := reader.Open(); err != nil {
		panic(err)
	}
	// writer := &tl.NullWriter{}
	writer, _ := tlcsv.NewWriter("tmp")
	if err := writer.Open(); err != nil {
		panic(err)
	}
	defer writer.Close()
	copier, err := copier.NewCopier(reader, writer, copier.Options{})
	if err != nil {
		panic(err)
	}
	copier.AddExtension(NewRouteGeometryBuilder())
	copier.AddExtension(NewRouteStopBuilder())
	copier.AddExtension(NewRouteHeadwayBuilder())
	copier.AddExtension(NewConvexHullBuilder())
	copier.AddExtension(NewOnestopIDBuilder())
	result := copier.Copy()
	if result.WriteError != nil {
		t.Fatal(result.WriteError)
	}
}
