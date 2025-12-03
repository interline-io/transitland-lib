package rules

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// StopTimeSequenceCheck checks that all sequences stop_time sequences in a trip are valid.
// This should be split into multiple validators.
type StopTimeSequenceCheck struct{}

// Validate .
func (e *StopTimeSequenceCheck) Validate(ent tt.Entity) []error {
	trip, ok := ent.(*gtfs.Trip)
	if !ok {
		return nil
	}
	// Use existing validator.
	var errs = ValidateStopTimes(trip.StopTimes)
	return errs
}

// hasTimeWindow returns true if the stop_time has GTFS-Flex time windows defined.
// Time windows are mutually exclusive with arrival_time/departure_time.
func hasTimeWindow(st gtfs.StopTime) bool {
	return (st.StartPickupDropOffWindow.Valid && st.StartPickupDropOffWindow.Val > 0) ||
		(st.EndPickupDropOffWindow.Valid && st.EndPickupDropOffWindow.Val > 0)
}

// ValidateStopTimes checks if the trip follows GTFS rules, including GTFS-Flex extensions.
func ValidateStopTimes(stoptimes []gtfs.StopTime) []error {
	errs := []error{}

	// 1. Check has >= 2 stop_times
	if len(stoptimes) == 0 {
		errs = append(errs, causes.NewEmptyTripError(len(stoptimes)))
		return errs // assumes >= 1 below
	}
	if len(stoptimes) < 2 {
		errs = append(errs, causes.NewEmptyTripError(len(stoptimes)))
	}

	// 2. First stop validation: Must have departure_time OR time window
	firstSt := stoptimes[0]
	if firstSt.DepartureTime.Int() <= 0 && !hasTimeWindow(firstSt) {
		errs = append(errs, causes.NewSequenceError("departure_time", "missing on first stop (required unless time window present)"))
	}

	// 3. Last stop validation: Must have arrival_time OR time window
	lastSt := stoptimes[len(stoptimes)-1]
	if lastSt.ArrivalTime.Int() <= 0 && !hasTimeWindow(lastSt) {
		errs = append(errs, causes.NewSequenceError("arrival_time", "missing on last stop (required unless time window present)"))
	}

	// Initialize tracking variables
	lastDist := stoptimes[0].ShapeDistTraveled
	lastScheduledTime := stoptimes[0].DepartureTime // Track time only for scheduled stops
	lastSequence := stoptimes[0].StopSequence

	// 4-6. Validate stop sequences, time progression, and shape distances
	for _, st := range stoptimes[1:] {
		// 4. Stop sequence validation: No duplicates, must increase
		if st.StopSequence == lastSequence {
			errs = append(errs, causes.NewSequenceError("stop_sequence", st.StopSequence.String()))
		} else {
			lastSequence = st.StopSequence
		}

		// 5. Time progression validation (only for scheduled stops, skip flex stops)
		if !hasTimeWindow(st) {
			// This is a scheduled stop with arrival/departure times
			if st.ArrivalTime.Int() > 0 && lastScheduledTime.Int() > 0 && st.ArrivalTime.Int() < lastScheduledTime.Int() {
				errs = append(errs, causes.NewSequenceError("arrival_time", st.ArrivalTime.String()))
			}
			if st.DepartureTime.Int() > 0 && st.ArrivalTime.Int() > 0 && st.DepartureTime.Int() < st.ArrivalTime.Int() {
				errs = append(errs, causes.NewSequenceError("departure_time", st.DepartureTime.String()))
			}
			// Update last scheduled time for next comparison
			if st.DepartureTime.Int() > 0 {
				lastScheduledTime = st.DepartureTime
			} else if st.ArrivalTime.Int() > 0 {
				lastScheduledTime = st.ArrivalTime
			}
		}
		// else: Flex stop with time window - skip time progression validation

		// 6. Shape distance validation: Must increase when present
		if st.ShapeDistTraveled.Valid && lastDist.Valid && st.ShapeDistTraveled.Val < lastDist.Val {
			errs = append(errs, causes.NewSequenceError("shape_dist_traveled", st.ShapeDistTraveled.String()))
		}
		if st.ShapeDistTraveled.Valid {
			lastDist = st.ShapeDistTraveled
		}
	}

	return errs
}
