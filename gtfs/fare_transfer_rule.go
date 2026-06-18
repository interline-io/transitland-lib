package gtfs

import (
	"fmt"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// FareTransferRule fare_transfer_rules.txt
type FareTransferRule struct {
	FromLegGroupID      tt.String `target:"fare_leg_rules.txt:leg_group_id" standardized_sort:"1"`
	ToLegGroupID        tt.String `target:"fare_leg_rules.txt:leg_group_id" standardized_sort:"2"`
	TransferCount       tt.Int    `range:"-1," standardized_sort:"4"`
	DurationLimit       tt.Int    `range:"0," standardized_sort:"5"`
	DurationLimitType   tt.Int    `enum:"0,1,2,3"`
	FareTransferType    tt.Int    `csv:",required" enum:"0,1,2"`
	FareProductID       tt.String `target:"fare_products.txt:fare_product_id" standardized_sort:"3"`
	FilterFareProductID tt.String `target:"fare_products.txt:fare_product_id"` // proposed extension
	tt.BaseEntity
}

func (ent *FareTransferRule) Filename() string {
	return "fare_transfer_rules.txt"
}

func (ent *FareTransferRule) TableName() string {
	return "gtfs_fare_transfer_rules"
}

func (ent *FareTransferRule) ConditionalErrors() (errs []error) {
	// Check disallowed value for transer_count
	if ent.TransferCount.Valid && ent.TransferCount.Val == 0 {
		errs = append(errs, causes.NewInvalidFieldError("transfer_count", fmt.Sprintf("%d", ent.TransferCount.Val), fmt.Errorf("must be -1 or greater than 0")))
	}

	// transfer_count
	legGroupsValidEqual := ent.FromLegGroupID.Valid && ent.FromLegGroupID.Val == ent.ToLegGroupID.Val
	if ent.TransferCount.Valid {
		if !legGroupsValidEqual {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("transfer_count", fmt.Sprintf("%d", ent.TransferCount.Val), "requires from_leg_group_id == to_leg_group_id"))
		}
	} else if legGroupsValidEqual {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("transfer_count"))
	}

	// duration_limit, duration_limit_type
	if ent.DurationLimitType.Valid {
		if !ent.DurationLimit.Valid {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("duration_limit", ent.DurationLimitType.String(), "duration_limit_type requires duration_limit to be present"))
		}
	} else if ent.DurationLimit.Valid {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("duration_limit"))
	}
	return errs
}

func (ent *FareTransferRule) DuplicateKey() string {
	return fmt.Sprintf(
		"fare_product_id:'%s' from_leg_group_id:'%s' to_leg_group_id:'%s' filter_fare_product_id:'%s' transfer_count:%d duration_limit:%d",
		ent.FareProductID.Val,
		ent.FromLegGroupID.Val,
		ent.ToLegGroupID.Val,
		ent.FilterFareProductID.Val,
		ent.TransferCount.Val,
		ent.DurationLimit.Val,
	)
}
