package tl

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tt"
)

// FareProduct fare_products.txt
type FareProduct struct {
	FareProductID   String
	FareProductName String
	Amount          Float
	Currency        String
	DurationStart   Int   // proposed extension
	DurationAmount  Float // proposed extension
	DurationUnit    Int   // proposed extension
	DurationType    Int   // proposed extension
	RiderCategoryID Key   // proposed extension
	FareContainerID Key   // proposed extension
	BaseEntity
}

func (ent *FareProduct) String() string {
	return fmt.Sprintf(
		"<fare_product fare_product_id:%s rider_category_id:%s fare_container_id:%s amount:%0.2f>",
		ent.FareProductID.Val,
		ent.RiderCategoryID.Val,
		ent.FareContainerID.Val,
		ent.Amount.Val,
	)
}

func (ent *FareProduct) EntityID() string {
	return ent.FareProductID.Val
}

func (ent *FareProduct) Filename() string {
	return "fare_products.txt"
}

func (ent *FareProduct) TableName() string {
	return "gtfs_fare_products"
}

func (ent *FareProduct) UpdateKeys(emap *EntityMap) error {
	if ent.FareContainerID.Val != "" {
		if fkid, ok := emap.Get("fare_containers.txt", ent.FareContainerID.Val); ok {
			ent.FareContainerID.Val = fkid
		} else {
			return causes.NewInvalidReferenceError("fare_container_id", ent.FareContainerID.Val)
		}
	}
	if ent.RiderCategoryID.Val != "" {
		if fkid, ok := emap.Get("rider_categories.txt", ent.RiderCategoryID.Val); ok {
			ent.RiderCategoryID.Val = fkid
			ent.RiderCategoryID.Valid = true
		} else {
			return causes.NewInvalidReferenceError("rider_category_id", ent.RiderCategoryID.Val)
		}
	}
	return nil
}

func (ent *FareProduct) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, tt.CheckPresent("fare_product_id", ent.FareProductID.Val)...)

	// amount
	if !ent.Amount.Valid {
		errs = append(errs, causes.NewRequiredFieldError("amount"))
	}

	// currency
	errs = append(errs, tt.CheckPresent("currency", ent.Currency.Val)...)
	errs = append(errs, tt.CheckCurrency("currency", ent.Currency.Val)...)

	// duration_start, duration_amount, duration_unit, duration_type
	errs = append(errs, tt.CheckInsideRangeInt("duration_start", ent.DurationStart.Val, 0, 1)...)
	errs = append(errs, tt.CheckPositive("duration_amount", ent.DurationAmount.Val)...)
	errs = append(errs, tt.CheckInsideRangeInt("duration_unit", ent.DurationUnit.Val, 0, 6)...)
	if ent.DurationType.Valid {
		errs = append(errs, tt.CheckInsideRangeInt("duration_type", ent.DurationType.Val, 1, 2)...)
	}
	if ent.DurationAmount.Valid && !ent.DurationType.Valid {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("duration_type"))
	}
	if ent.DurationType.Valid && !ent.DurationAmount.Valid {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("duration_amount"))
	}
	return errs
}
