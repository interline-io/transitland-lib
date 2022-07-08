package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl"
)

// DuplicateFareRuleError reports when multiple FareRules have the same (route_id,origin_id,_destination_id,contains_id)
type DuplicateFareRuleError struct {
	RouteID       string
	OriginID      string
	DestinationID string
	ContainsID    string
	bc
}

func (e *DuplicateFareRuleError) Error() string {
	return fmt.Sprintf(
		"fare_rule with route_id '%s' origin_id '%s' destination_id '%s' and agency_id '%s' is duplicated",
		e.RouteID,
		e.OriginID,
		e.DestinationID,
		e.ContainsID,
	)
}

// DuplicateFareRuleCheck checks for DuplicateFareRuleErrors.
type DuplicateFareRuleCheck struct {
	rules map[string]int
}

// Validate .
func (e *DuplicateFareRuleCheck) Validate(ent tl.Entity) []error {
	v, ok := ent.(*tl.FareRule)
	if !ok {
		return nil
	}
	if e.rules == nil {
		e.rules = map[string]int{}
	}
	key := v.RouteID.Key + ":" + v.OriginID.Key + ":" + v.DestinationID.Key + ":" + v.ContainsID.Key
	if _, ok := e.rules[key]; ok {
		return []error{&DuplicateFareRuleError{
			RouteID:       v.RouteID.Key,
			OriginID:      v.OriginID.Key,
			DestinationID: v.DestinationID.Key,
			ContainsID:    v.ContainsID.Key,
		}}
	}
	e.rules[key]++
	return nil
}
