package extract

import (
	"testing"

	"github.com/interline-io/gotransit"
)

func TestSetterFilter_Filter(t *testing.T) {
	stop := &gotransit.Stop{StopID: "abc"}
	route := &gotransit.Route{RouteID: "foo"}
	emap := gotransit.NewEntityMap()
	tx := NewSetterFilter()
	tx.AddValue(stop.Filename(), stop.EntityID(), "stop_name", "test")
	tx.AddValue(route.Filename(), route.EntityID(), "route_type", "1000")
	tx.Filter(stop, emap)
	tx.Filter(route, emap)
	if stop.StopName != "test" {
		t.Errorf("got %s expect %s", stop.StopName, "test")
	}
	if route.RouteType != 1000 {
		t.Errorf("got %d expect %d", route.RouteType, 1000)
	}
}
