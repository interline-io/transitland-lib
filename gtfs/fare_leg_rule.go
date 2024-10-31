package gtfs

import (
	"github.com/interline-io/transitland-lib/tt"
)

// FareLegRule fare_leg_rules.txt
type FareLegRule struct {
	LegGroupID    tt.String
	FromAreaID    tt.String `target:"areas.txt"`
	ToAreaID      tt.String `target:"areas.txt"`
	NetworkID     tt.String `target:"routes.txt:network_id"`
	FareProductID tt.String `csv:",required" target:"fare_products.txt:fare_product_id"`
	TransferOnly  tt.Int    `enum:"0,1"` // interline ext
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
