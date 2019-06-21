package extract

import (
	"testing"

	"github.com/interline-io/gotransit"
)

func TestSetterFilter(t *testing.T) {
	stop := &gotransit.Stop{StopID: "abc"}
	emap := gotransit.NewEntityMap()

	tx := newSetterFilter()
	tx.nodes[*entityNode(stop)] = map[string]string{"stop_name": "test"}
	tx.Filter(stop, emap)
	if stop.StopName != "test" {
		t.Errorf("got %s expect %s", stop.StopName, "test")
	}
}
