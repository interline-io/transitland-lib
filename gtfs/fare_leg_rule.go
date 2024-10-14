package gtfs

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tt"
)

// FareLegRule fare_leg_rules.txt
type FareLegRule struct {
	LegGroupID    tt.String
	FromAreaID    tt.String `target:"areas.txt"`
	ToAreaID      tt.String `target:"areas.txt"`
	NetworkID     tt.String `target:"routes.txt:network_id"`
	FareProductID tt.String `target:"fare_products.txt"`
	TransferOnly  tt.Int    // interline ext
	tt.BaseEntity
}

func (ent *FareLegRule) String() string {
	return fmt.Sprintf(
		"<fare_leg_rule leg_group_id:%s from_area_id:%s to_area_id:%s network_id:%s product:%s transfer_only:%d>",
		ent.LegGroupID.Val,
		ent.FromAreaID.Val,
		ent.ToAreaID.Val,
		ent.NetworkID.Val,
		ent.FareProductID.Val,
		ent.TransferOnly.Val,
	)
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

func (ent *FareLegRule) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	// Final spec: leg_group_id is optional
	// errs = append(errs, tt.CheckPresent("leg_group_id", ent.LegGroupID.Val)...)
	errs = append(errs, tt.CheckPresent("fare_product_id", ent.FareProductID.Val)...)
	errs = append(errs, tt.CheckInsideRangeInt("transfer_only", ent.TransferOnly.Val, 0, 1)...)
	return errs
}
