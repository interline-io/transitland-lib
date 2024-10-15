package extract

import (
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

func TestSetterFilter_Filter(t *testing.T) {
	stop := &gtfs.Stop{StopID: tt.NewString("abc")}
	route := &gtfs.Route{RouteID: tt.NewString("foo")}
	emap := tt.NewEntityMap()
	tx := NewSetterFilter()
	tx.AddValue(stop.Filename(), stop.EntityID(), "stop_name", "test")
	tx.AddValue(route.Filename(), route.EntityID(), "route_type", "1000")
	tx.Filter(stop, emap)
	tx.Filter(route, emap)
	if stop.StopName.Val != "test" {
		t.Errorf("got %s expect %s", stop.StopName, "test")
	}
	if route.RouteType.Val != 1000 {
		t.Errorf("got %d expect %d", route.RouteType.Val, 1000)
	}
}
