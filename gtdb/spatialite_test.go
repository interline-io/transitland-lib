package gtdb

import (
	"testing"
)

func TestSpatiaLiteAdapter(t *testing.T) {
	if adapter, ok := getTestAdapters()["SpatiaLiteAdapter-Memory"]; ok {
		testAdapter(t, adapter())
	} else {
		t.Skip("skipping SpatiaLiteAdapter-Memory adapter tests")
	}
}
