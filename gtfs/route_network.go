package gtfs

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tt"
)

type RouteNetwork struct {
	NetworkID tt.Key `target:"networks.txt"`
	RouteID   tt.Key `target:"routes.txt"`
	tt.BaseEntity
}

func (ent *RouteNetwork) Filename() string {
	return "route_networks.txt"
}

func (ent *RouteNetwork) TableName() string {
	return "gtfs_route_networks"
}

func (ent *RouteNetwork) DuplicateKey() string {
	return fmt.Sprintf(
		"network_id:'%s' route_id:'%s'",
		ent.NetworkID.Val,
		ent.RouteID.Val,
	)
}
