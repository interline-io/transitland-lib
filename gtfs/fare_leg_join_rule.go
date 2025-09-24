package gtfs

import (
	"github.com/interline-io/transitland-lib/tt"
)

// FareLegJoinRule fare_leg_join_rules.txt
type FareLegJoinRule struct {
	FromNetworkID tt.String `csv:",required" target:"networks.txt"`
	ToNetworkID   tt.String `csv:",required" target:"networks.txt"`
	FromStopID    tt.String `target:"stops.txt"`
	ToStopID      tt.String `target:"stops.txt"`
	tt.BaseEntity
}

// EntityID returns the ID or composite key.
func (ent *FareLegJoinRule) EntityID() string {
	return entID(ent.ID, ent.FromNetworkID.Val+"_"+ent.ToNetworkID.Val+"_"+ent.FromStopID.Val+"_"+ent.ToStopID.Val)
}

// EntityKey returns the GTFS identifier.
func (ent *FareLegJoinRule) EntityKey() string {
	return ent.FromNetworkID.Val + "_" + ent.ToNetworkID.Val + "_" + ent.FromStopID.Val + "_" + ent.ToStopID.Val
}

// Filename fare_leg_join_rules.txt
func (ent *FareLegJoinRule) Filename() string {
	return "fare_leg_join_rules.txt"
}

// TableName gtfs_fare_leg_join_rules
func (ent *FareLegJoinRule) TableName() string {
	return "gtfs_fare_leg_join_rules"
}
