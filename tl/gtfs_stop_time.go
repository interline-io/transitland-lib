package tl

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/interline-io/transitland-lib/causes"
)

// StopTime stop_times.txt
type StopTime struct {
	TripID            string  `csv:"trip_id"`
	ArrivalTime       int     `csv:"arrival_time" `
	DepartureTime     int     `csv:"departure_time" `
	StopID            string  `csv:"stop_id" required:"true"`
	StopSequence      int     `csv:"stop_sequence" required:"true" min:"0"`
	StopHeadsign      string  `csv:"stop_headsign"`
	PickupType        int     `csv:"pickup_type" min:"0" max:"3"`
	DropOffType       int     `csv:"drop_off_type" min:"0" max:"3"`
	ShapeDistTraveled float64 `csv:"shape_dist_traveled" min:"0"`
	Timepoint         int     `csv:"timepoint" min:"-1" max:"1"` // -1 for empty
	Interpolated      int     // interpolated times: 0 for provided, 1 interpolated // TODO: 1 for shape, 2 for straight-line
	BaseEntity
}

// EntityID returns nothing.
func (ent *StopTime) EntityID() string {
	return ""
}

// Errors for this Entity.
func (ent *StopTime) Errors() []error {
	// No reflection
	errs := []error{}
	errs = append(errs, ent.BaseEntity.loadErrors...)
	if len(ent.TripID) == 0 {
		errs = append(errs, causes.NewRequiredFieldError("trip_id"))
	}
	if len(ent.StopID) == 0 {
		errs = append(errs, causes.NewRequiredFieldError("stop_id"))
	}
	if ent.StopSequence < 0 {
		errs = append(errs, causes.NewInvalidFieldError("stop_sequence", "", fmt.Errorf("negative stop_sequence: %d", ent.StopSequence)))
	}
	if ent.PickupType < 0 || ent.PickupType > 3 {
		errs = append(errs, causes.NewInvalidFieldError("pickup_type", "", fmt.Errorf("pickup_type out of bounds: %d", ent.PickupType)))
	}
	if ent.DropOffType < 0 || ent.DropOffType > 3 {
		errs = append(errs, causes.NewInvalidFieldError("drop_off_type", "", fmt.Errorf("drop_off_type out of bounds: %d", ent.DropOffType)))
	}
	if ent.ShapeDistTraveled < 0 && ent.ShapeDistTraveled != -1.0 {
		errs = append(errs, causes.NewInvalidFieldError("shape_dist_traveled", "", fmt.Errorf("negative shape_dist_traveled: %f", ent.ShapeDistTraveled)))
	}
	if ent.Timepoint < -1 || ent.Timepoint > 1 {
		errs = append(errs, causes.NewInvalidFieldError("timepoint", "", fmt.Errorf("timepoint out of bounds: %d", ent.Timepoint)))
	}
	// Other errors
	at, dt := ent.ArrivalTime, ent.DepartureTime
	if at != 0 && dt != 0 && at > dt {
		errs = append(errs, causes.NewInvalidFieldError("departure_time", "", fmt.Errorf("departure_time '%d' must come after arrival_time '%d'", dt, at)))
	}
	return errs
}

// Filename stop_times.txt
func (ent *StopTime) Filename() string {
	return "stop_times.txt"
}

// TableName gtfs_stop_times
func (ent *StopTime) TableName() string {
	return "gtfs_stop_times"
}

// UpdateKeys updates Entity references.
func (ent *StopTime) UpdateKeys(emap *EntityMap) error {
	if tripID, ok := emap.GetEntity(&Trip{TripID: ent.TripID}); ok {
		ent.TripID = tripID
	} else {
		return causes.NewInvalidReferenceError("trip_id", ent.TripID)
	}
	if stopID, ok := emap.GetEntity(&Stop{StopID: ent.StopID}); ok {
		ent.StopID = stopID
	} else {
		return causes.NewInvalidReferenceError("stop_id", ent.StopID)
	}
	return nil
}

// GetString returns the string representation of an field.
func (ent *StopTime) GetString(key string) (string, error) {
	v := ""
	switch key {
	case "trip_id":
		v = ent.TripID
	case "stop_headsign":
		v = ent.StopHeadsign
	case "stop_id":
		v = ent.StopID
	case "arrival_time":
		v = SecondsToString(ent.ArrivalTime)
	case "departure_time":
		v = SecondsToString(ent.DepartureTime)
	case "stop_sequence":
		v = strconv.Itoa(ent.StopSequence)
	case "pickup_type":
		v = strconv.Itoa(ent.PickupType)
	case "drop_off_type":
		v = strconv.Itoa(ent.DropOffType)
	case "shape_dist_traveled":
		if ent.ShapeDistTraveled >= 0 {
			v = fmt.Sprintf("%0.5f", ent.ShapeDistTraveled)
		}
	case "timepoint":
		if ent.Timepoint > -1 {
			v = strconv.Itoa(ent.Timepoint)
		}
	default:
		return v, errors.New("unknown key")
	}
	return v, nil
}

// SetString provides a fast, non-reflect loading path.
func (ent *StopTime) SetString(key, value string) error {
	var perr error
	hi := value
	switch key {
	case "trip_id":
		ent.TripID = hi
	case "stop_headsign":
		ent.StopHeadsign = hi
	case "stop_id":
		ent.StopID = hi
	case "arrival_time":
		if hi == "" {
			ent.ArrivalTime = -1
		} else if s, err := StringToSeconds(hi); err != nil {
			perr = causes.NewFieldParseError("arrival_time", hi)
		} else {
			ent.ArrivalTime = s
		}
	case "departure_time":
		if hi == "" {
			ent.DepartureTime = -1
		} else if s, err := StringToSeconds(hi); err != nil {
			perr = causes.NewFieldParseError("departure_time", hi)
		} else {
			ent.DepartureTime = s
		}
	case "stop_sequence":
		if a, err := strconv.Atoi(hi); err != nil {
			perr = causes.NewFieldParseError("stop_sequence", hi)
		} else {
			ent.StopSequence = a
		}
	case "pickup_type":
		if len(hi) == 0 {
			ent.PickupType = 0
		} else if a, err := strconv.Atoi(hi); err != nil {
			perr = causes.NewFieldParseError("pickup_type", hi)
		} else {
			ent.PickupType = a
		}
	case "drop_off_type":
		if len(hi) == 0 {
			ent.DropOffType = 0
		} else if a, err := strconv.Atoi(hi); err != nil {
			perr = causes.NewFieldParseError("drop_off_type", hi)
		} else {
			ent.DropOffType = a
		}
	case "shape_dist_traveled":
		if len(hi) == 0 {
			ent.ShapeDistTraveled = -1.0
		} else if a, err := strconv.ParseFloat(hi, 64); err != nil {
			perr = causes.NewFieldParseError("shape_dist_traveled", hi)
		} else {
			ent.ShapeDistTraveled = a
		}
	case "timepoint":
		// special use -1 for empty timepoint value
		if len(hi) == 0 {
			ent.Timepoint = -1
		} else if a, err := strconv.Atoi(hi); err != nil {
			perr = causes.NewFieldParseError("timepoint", hi)
		} else {
			ent.Timepoint = a
		}
	default:
		ent.SetExtra(key, hi)
	}
	return perr
}

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
	if stoptimes[len(stoptimes)-1].ArrivalTime <= 0 {
		errs = append(errs, causes.NewSequenceError("arrival_time", ""))
	}
	lastDist := stoptimes[0].ShapeDistTraveled
	lastTime := stoptimes[0].DepartureTime
	lastSequence := stoptimes[0].StopSequence
	for _, st := range stoptimes[1:] {
		// Ensure we do not have duplicate StopSequennce
		if st.StopSequence == lastSequence {
			errs = append(errs, causes.NewSequenceError("stop_sequence", strconv.Itoa(st.StopSequence)))
		} else {
			lastSequence = st.StopSequence
		}
		// Ensure the arrows of time are pointing towards the future.
		if st.ArrivalTime > 0 && st.ArrivalTime < lastTime {
			errs = append(errs, causes.NewSequenceError("arrival_time", strconv.Itoa(st.ArrivalTime)))
		} else if st.DepartureTime > 0 && st.DepartureTime < st.ArrivalTime {
			errs = append(errs, causes.NewSequenceError("departure_time", strconv.Itoa(st.DepartureTime)))
		} else if st.DepartureTime > 0 {
			lastTime = st.DepartureTime
		}
		if st.ShapeDistTraveled > 0 && st.ShapeDistTraveled < lastDist {
			errs = append(errs, causes.NewSequenceError("shape_dist_traveled", fmt.Sprintf("%f", st.ShapeDistTraveled)))
		} else if st.ShapeDistTraveled > 0 {
			lastDist = st.ShapeDistTraveled
		}
	}
	return errs
}
