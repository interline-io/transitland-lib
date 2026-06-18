package gtfs

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tt"
)

type RouteNetwork struct {
	NetworkID tt.Key `csv:",required" target:"networks.txt" standardized_sort:"1"`
	RouteID   tt.Key `csv:",required" target:"routes.txt" standardized_sort:"2"`
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
