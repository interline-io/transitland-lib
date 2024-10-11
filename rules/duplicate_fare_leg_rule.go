package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tt"
)

// DuplicateFareLegRule reports when multiple FareLegRules have the same unique values.
type DuplicateFareLegRuleError struct {
	FareProductID string
	NetworkID     string
	FromAreaID    string
	ToAreaID      string
	bc
}

func (e *DuplicateFareLegRuleError) Error() string {
	return fmt.Sprintf(
		"fare_leg_rule with fare_product_id '%s' network_id '%s' from_area_id '%s' and to_area_id '%s' is duplicated",
		e.FareProductID,
		e.NetworkID,
		e.FromAreaID,
		e.ToAreaID,
	)
}

// DuplicateFareRuleCheck checks for DuplicateFareLegRuleErrors.
type DuplicateFareLegRuleCheck struct {
	vals map[string]int
}

func (e *DuplicateFareLegRuleCheck) Validate(ent tt.Entity) []error {
	v, ok := ent.(*tl.FareLegRule)
	if !ok {
		return nil
	}
	if e.vals == nil {
		e.vals = map[string]int{}
	}
	err := DuplicateFareLegRuleError{
		FareProductID: v.FareProductID.Val,
		NetworkID:     v.NetworkID.Val,
		FromAreaID:    v.FromAreaID.Val,
		ToAreaID:      v.ToAreaID.Val,
	}
	key := fmt.Sprintf("%s:%s:%s:%s", err.FareProductID, err.NetworkID, err.FromAreaID, err.ToAreaID)
	if _, ok := e.vals[key]; ok {
		return []error{&err}
	}
	e.vals[key]++
	return nil
}
