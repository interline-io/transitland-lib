package tl

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tt"
)

// FareLegRule fare_leg_rules.txt
type FareLegRule struct {
	LegGroupID    String
	FromAreaID    String
	ToAreaID      String
	NetworkID     String
	FareProductID String
	TransferOnly  tt.Int // interline ext
	BaseEntity
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

func (ent *FareLegRule) UpdateKeys(emap *EntityMap) error {
	if ent.FromAreaID.Val != "" {
		if fkid, ok := emap.Get("areas.txt", ent.FromAreaID.Val); ok {
			ent.FromAreaID = tt.NewString(fkid)
		} else {
			return causes.NewInvalidReferenceError("from_area_id", ent.FromAreaID.Val)
		}
	}
	if ent.ToAreaID.Val != "" {
		if fkid, ok := emap.Get("areas.txt", ent.ToAreaID.Val); ok {
			ent.ToAreaID = tt.NewString(fkid)
		} else {
			return causes.NewInvalidReferenceError("to_area_id", ent.ToAreaID.Val)
		}
	}
	// Check fare network
	if ent.NetworkID.Val == "" {
		// ok
	} else if fkid2, ok2 := emap.Get("routes.txt:network_id", ent.NetworkID.Val); ok2 {
		ent.NetworkID = tt.NewString(fkid2)
	} else {
		return causes.NewInvalidReferenceError("network_id", ent.NetworkID.Val)
	}
	// Check fare product
	if ent.FareProductID.Val != "" {
		if fkid, ok := emap.Get("fare_products.txt", ent.FareProductID.Val); ok {
			ent.FareProductID = tt.NewString(fkid)
		} else {
			return causes.NewInvalidReferenceError("fare_product_id", ent.FareProductID.Val)
		}
	}
	return nil
}

func (ent *FareLegRule) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	// Final spec: leg_group_id is optional
	// errs = append(errs, tt.CheckPresent("leg_group_id", ent.LegGroupID.Val)...)
	errs = append(errs, tt.CheckPresent("fare_product_id", ent.FareProductID.Val)...)
	errs = append(errs, tt.CheckInsideRangeInt("transfer_only", ent.TransferOnly.Val, 0, 1)...)
	return errs
}
