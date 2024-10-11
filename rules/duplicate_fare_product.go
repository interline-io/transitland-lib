package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tt"
)

// DuplicateFareProduct reports when multiple FareProducts have the same unique values.
type DuplicateFareProductError struct {
	FareProductID   string
	RiderCategoryID string
	FareMediaID     string
	bc
}

func (e *DuplicateFareProductError) Error() string {
	return fmt.Sprintf(
		"fare_product_rule with fare_product_id '%s' rider_category_id '%s' and fare_media_id '%s' is duplicated",
		e.FareProductID,
		e.RiderCategoryID,
		e.FareMediaID,
	)
}

// DuplicateFareRuleCheck checks for DuplicateFareProductErrors.
type DuplicateFareProductCheck struct {
	vals map[string]int
}

func (e *DuplicateFareProductCheck) Validate(ent tt.Entity) []error {
	v, ok := ent.(*tl.FareProduct)
	if !ok {
		return nil
	}
	if e.vals == nil {
		e.vals = map[string]int{}
	}
	err := DuplicateFareProductError{
		FareProductID:   v.FareProductID.Val,
		RiderCategoryID: v.RiderCategoryID.Val,
		FareMediaID:     v.FareMediaID.Val,
	}
	key := fmt.Sprintf("%s:%s:%s", err.FareProductID, err.RiderCategoryID, err.FareMediaID)
	if _, ok := e.vals[key]; ok {
		return []error{&err}
	}
	e.vals[key]++
	return nil
}
