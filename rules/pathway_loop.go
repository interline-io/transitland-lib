package rules

import "github.com/interline-io/transitland-lib/tl"

type PathwayLoopError struct{ bc }

type PathwayLoopCheck struct{}

func (e *PathwayLoopCheck) Validate(ent tl.Entity) []error {
	if v, ok := ent.(*tl.Pathway); ok {
		if v.FromStopID == v.ToStopID {
			err := PathwayLoopError{}
			err.Field = "from_stop_id"
			err.Message = "from_stop_id cannot equal to_stop_id"
			return []error{&err}
		}
	}
	return nil
}
