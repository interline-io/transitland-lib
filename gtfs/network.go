package gtfs

import "github.com/interline-io/transitland-lib/tt"

type Network struct {
	NetworkID   tt.String `csv:",required"`
	NetworkName tt.String
	tt.BaseEntity
}

func (ent *Network) EntityKey() string {
	return ent.NetworkID.Val
}

func (ent *Network) EntityID() string {
	return entID(ent.ID, ent.NetworkID.Val)
}

func (ent *Network) Filename() string {
	return "networks.txt"
}

func (ent *Network) TableName() string {
	return "gtfs_networks"
}
