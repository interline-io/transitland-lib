package rules

import "github.com/interline-io/transitland-lib/tl"

// StopTimeSequenceCheck checks that all sequences stop_time sequences in a trip are valid.
// This should be split into multiple validators.
type StopTimeSequenceCheck struct{}

// Validate .
func (e *StopTimeSequenceCheck) Validate(ent tl.Entity) []error {
	trip, ok := ent.(*tl.Trip)
	if !ok {
		return nil
	}
	// Use existing validator.
	var errs = tl.ValidateStopTimes(trip.StopTimes)
	return errs
}
