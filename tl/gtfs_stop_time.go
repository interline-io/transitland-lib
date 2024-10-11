package tl

import (
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tt"
)

// ValidateStopTimes checks if the trip follows GTFS rules.
func ValidateStopTimes(stoptimes []StopTime) []error {
	errs := []error{}
	if len(stoptimes) == 0 {
		errs = append(errs, causes.NewEmptyTripError(len(stoptimes)))
		return errs // assumes >= 1 below
	}
	if len(stoptimes) < 2 {
		errs = append(errs, causes.NewEmptyTripError(len(stoptimes)))
	}
	if lastSt := stoptimes[len(stoptimes)-1]; lastSt.ArrivalTime.Int() <= 0 {
		errs = append(errs, causes.NewSequenceError("arrival_time", lastSt.ArrivalTime.String()))
	}
	lastDist := stoptimes[0].ShapeDistTraveled
	lastTime := stoptimes[0].DepartureTime
	lastSequence := stoptimes[0].StopSequence
	for _, st := range stoptimes[1:] {
		// Ensure we do not have duplicate StopSequennce
		if st.StopSequence == lastSequence {
			errs = append(errs, causes.NewSequenceError("stop_sequence", tt.TryCsv(st.StopSequence)))
		} else {
			lastSequence = st.StopSequence
		}
		// Ensure the arrows of time are pointing towards the future.
		if st.ArrivalTime.Int() > 0 && st.ArrivalTime.Int() < lastTime.Int() {
			errs = append(errs, causes.NewSequenceError("arrival_time", st.ArrivalTime.String()))
		} else if st.DepartureTime.Int() > 0 && st.DepartureTime.Int() < st.ArrivalTime.Int() {
			errs = append(errs, causes.NewSequenceError("departure_time", st.DepartureTime.String()))
		} else if st.DepartureTime.Int() > 0 {
			lastTime = st.DepartureTime
		}
		if st.ShapeDistTraveled.Valid && st.ShapeDistTraveled.Val < lastDist.Val {
			errs = append(errs, causes.NewSequenceError("shape_dist_traveled", st.ShapeDistTraveled.String()))
		} else if st.ShapeDistTraveled.Valid {
			lastDist = st.ShapeDistTraveled
		}
	}
	return errs
}
