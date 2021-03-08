package rules

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
)

// DuplicateFareRuleCheck checks for fare_rules that are effectively identical.
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
	key := v.RouteID.Key + ":" + v.OriginID + ":" + v.DestinationID + ":" + v.ContainsID
	if _, ok := e.rules[key]; ok {
		return []error{causes.NewValidationWarning("origin_id", "duplicate fare_rule")}
	}
	e.rules[key]++
	return nil
}
