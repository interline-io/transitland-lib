package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

type RouteNetworkIDFilter struct{}

func (e *RouteNetworkIDFilter) Filter(ent tt.Entity, emap *tt.EntityMap) error {
	if v, ok := ent.(*gtfs.Route); ok {
		v.NetworkID = tt.String{}
	}
	return nil
}

func (e *RouteNetworkIDFilter) Expand(ent tt.Entity, emap *tt.EntityMap) ([]tt.Entity, bool, error) {
	v, ok := ent.(*gtfs.Route)
	if !ok {
		return nil, false, nil
	}
	if !v.NetworkID.Valid {
		return nil, false, nil
	}
	fmt.Println("RN:", v.NetworkID, v.RouteID)
	var ret []tt.Entity
	ret = append(ret, ent)
	if _, ok := emap.Get("networks.txt", v.NetworkID.Val); !ok {
		n := gtfs.Network{}
		n.NetworkID.Set(v.NetworkID.Val)
		ret = append(ret, &n)
		fmt.Println("CREATING NETWORK:", n)
	}
	rn := gtfs.RouteNetwork{}
	rn.NetworkID.Set(v.NetworkID.Val)
	rn.RouteID.Set(v.RouteID.Val)
	ret = append(ret, &rn)
	fmt.Println("CREATING ROUTE NETWORK:", rn)
	return ret, true, nil
}
