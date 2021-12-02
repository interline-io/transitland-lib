package rules

import "github.com/interline-io/transitland-lib/tl"

type PathwayTopologyError struct{ bc }

type PathwayTopologyCheck struct{}

func (e *PathwayTopologyCheck) AfterCopy(ent tl.Entity) []error {
	if v, ok := ent.(*tl.Pathway); ok {
		_ = v
	}
	return nil
}
