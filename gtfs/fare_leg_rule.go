package gtfs

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tt"
)

// FareLegRule fare_leg_rules.txt
type FareLegRule struct {
	LegGroupID           tt.String
	FromAreaID           tt.String `target:"areas.txt"`
	ToAreaID             tt.String `target:"areas.txt"`
	NetworkID            tt.String `target:"networks.txt"`
	FareProductID        tt.String `csv:",required" target:"fare_products.txt:fare_product_id"`
	FromTimeframeGroupID tt.String `target:"timeframes.txt:timeframe_group_id"`
	ToTimeframeGroupID   tt.String `target:"timeframes.txt:timeframe_group_id"`
	RulePriority         tt.Int    `range:"0,"`
	TransferOnly         tt.Int    `enum:"0,1"` // interline ext
	tt.BaseEntity
}

func (ent *FareLegRule) EntityID() string {
	return ent.LegGroupID.Val
}

func (ent *FareLegRule) Filename() string {
	return "fare_leg_rules.txt"
}

func (ent *FareLegRule) TableName() string {
	return "gtfs_fare_leg_rules"
}

func (ent *FareLegRule) GroupKey() (string, string) {
	return "leg_group_id", ent.LegGroupID.Val
}

func (ent *FareLegRule) DuplicateKey() string {
	key := fmt.Sprintf(
		"fare_product_id:'%s' network_id:'%s' from_area_id:'%s' to_area_id:'%s' from_timeframe_group_id:'%s' to_timeframe_group_id:'%s'",
		ent.FareProductID.Val,
		ent.NetworkID.Val,
		ent.FromAreaID.Val,
		ent.ToAreaID.Val,
		ent.FromTimeframeGroupID.Val,
		ent.ToTimeframeGroupID.Val,
	)
	return key
}
