package rules

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tlutil"
	"github.com/interline-io/transitland-lib/tt"
)

// StopTimeSequenceCheck checks that all sequences stop_time sequences in a trip are valid.
// This should be split into multiple validators.
type StopTimeSequenceCheck struct{}

// Validate .
func (e *StopTimeSequenceCheck) Validate(ent tt.Entity) []error {
	trip, ok := ent.(*tl.Trip)
	if !ok {
		return nil
	}
	// Use existing validator.
	var errs = tlutil.ValidateStopTimes(trip.StopTimes)
	return errs
}
