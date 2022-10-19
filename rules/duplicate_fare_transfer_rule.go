package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl"
)

// DuplicateFareTransferRule reports when multiple FareTransferRules have the same unique values.
type DuplicateFareTransferRuleError struct {
	FareProductID  string
	FromLegGroupID string
	ToLegGroupID   string
	TransferCount  int
	DurationLimit  int
	bc
}

func (e *DuplicateFareTransferRuleError) Error() string {
	return fmt.Sprintf(
		"fare_transfer_rule with fare_product_id '%s' from_leg_group_id '%s' to_leg_group_id '%s' transfer_count %d and duration_limit %d is duplicated",
		e.FareProductID,
		e.FromLegGroupID,
		e.ToLegGroupID,
		e.TransferCount,
		e.DurationLimit,
	)
}

// DuplicateFareRuleCheck checks for DuplicateFareTransferRuleErrors.
type DuplicateFareTransferRuleCheck struct {
	vals map[string]int
}

// Validate .
func (e *DuplicateFareTransferRuleCheck) Validate(ent tl.Entity) []error {
	v, ok := ent.(*tl.FareTransferRule)
	if !ok {
		return nil
	}
	if e.vals == nil {
		e.vals = map[string]int{}
	}
	err := DuplicateFareTransferRuleError{
		FareProductID:  v.FareProductID.Val,
		FromLegGroupID: v.FromLegGroupID.Val,
		ToLegGroupID:   v.ToLegGroupID.Val,
		TransferCount:  v.TransferCount.Val,
		DurationLimit:  v.DurationLimit.Val,
	}
	key := fmt.Sprintf("%s:%s:%s:%d:%d", err.FareProductID, err.FromLegGroupID, err.ToLegGroupID, err.TransferCount, err.DurationLimit)
	if _, ok := e.vals[key]; ok {
		return []error{&err}
	}
	e.vals[key]++
	return nil
}
