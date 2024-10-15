package gtfs

import (
	"fmt"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// FareTransferRule fare_transfer_rules.txt
type FareTransferRule struct {
	FromLegGroupID      tt.String `target:"fare_leg_rules.txt"`
	ToLegGroupID        tt.String `target:"fare_leg_rules.txt"`
	TransferCount       tt.Int
	DurationLimit       tt.Int
	DurationLimitType   tt.Int
	FareTransferType    tt.Int
	FareProductID       tt.String `target:"fare_products.txt"`
	FilterFareProductID tt.String `target:"fare_products.txt"` // proposed extension
	tt.BaseEntity
}

func (ent *FareTransferRule) String() string {
	return fmt.Sprintf(
		"<fare_transfer_rule from_leg_group_id:%s to_leg_group_id:%s product:%s duration_limit:%d duration_limit_type:%d fare_transfer_type:%d>",
		ent.FromLegGroupID.Val,
		ent.ToLegGroupID.Val,
		ent.FareProductID.Val,
		ent.DurationLimit.Val,
		ent.DurationLimitType.Val,
		ent.FareTransferType.Val,
	)
}

func (ent *FareTransferRule) Filename() string {
	return "fare_transfer_rules.txt"
}

func (ent *FareTransferRule) TableName() string {
	return "gtfs_fare_transfer_rules"
}

func (ent *FareTransferRule) Errors() (errs []error) {
	// Is optional in final spec
	// errs = append(errs, tt.CheckPresent("fare_product_id", ent.FareProductID.Val)...)

	// transfer_count
	legGroupsValidEqual := ent.FromLegGroupID.Valid && ent.FromLegGroupID.Val == ent.ToLegGroupID.Val
	if ent.TransferCount.Valid {
		if !legGroupsValidEqual {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("transfer_count", fmt.Sprintf("%d", ent.TransferCount.Val), "requires from_leg_group_id == to_leg_group_id"))
		}
		if ent.TransferCount.Val == 0 || ent.TransferCount.Val < -1 {
			errs = append(errs, causes.NewInvalidFieldError("transfer_count", fmt.Sprintf("%d", ent.TransferCount.Val), fmt.Errorf("must be -1 or greater than 0")))
		}
	} else if legGroupsValidEqual {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("transfer_count"))
	}

	// duration_limit, duration_limit_type
	if ent.DurationLimitType.Valid {
		if !ent.DurationLimit.Valid {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("duration_limit", tt.TryCsv(ent.DurationLimitType), "duration_limit_type requires duration_limit to be present"))
		}
		errs = append(errs, tt.CheckInsideRangeInt("duration_limit_type", int(ent.DurationLimitType.Val), 0, 3)...)
	} else if ent.DurationLimit.Valid {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("duration_limit"))
	}
	errs = append(errs, tt.CheckPositiveInt("duration_limit", ent.DurationLimit.Val)...)

	// fare_transfer_type
	if ent.FareTransferType.Valid {
		errs = append(errs, tt.CheckInsideRangeInt("fare_transfer_type", int(ent.FareTransferType.Val), 0, 2)...)
	} else {
		errs = append(errs, causes.NewRequiredFieldError("fare_transfer_type"))
	}

	return errs
}
