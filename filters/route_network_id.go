package filters

import (
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

type RouteNetworkIDFilter struct{}

func (e *RouteNetworkIDFilter) Expand(ent tt.Entity, emap *tt.EntityMap) ([]tt.Entity, bool, error) {
	// Check if route and has NetworkID set
	v, ok := ent.(*gtfs.Route)
	if !ok {
		return nil, false, nil
	}
	if !v.NetworkID.Valid {
		return nil, false, nil
	}
	// Expand into route + route_network + possible network
	var ret []tt.Entity
	ret = append(ret, ent)
	if _, ok := emap.Get("networks.txt", v.NetworkID.Val); !ok {
		n := gtfs.Network{}
		n.NetworkID.Set(v.NetworkID.Val)
		ret = append(ret, &n)
	}
	rn := gtfs.RouteNetwork{}
	rn.NetworkID.Set(v.NetworkID.Val)
	rn.RouteID.Set(v.RouteID.Val)
	ret = append(ret, &rn)
	return ret, true, nil
}

func (e *RouteNetworkIDFilter) Filter(ent tt.Entity, emap *tt.EntityMap) error {
	// Unset any set NetworkID
	if v, ok := ent.(*gtfs.Route); ok {
		v.NetworkID = tt.String{}
	}
	return nil
}
