package tl

import (
	"fmt"
	"strconv"

	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tt"
)

// StopTime stop_times.txt
type StopTime struct {
	TripID            string
	StopID            string `csv:",required" required:"true"`
	StopSequence      int    `csv:",required" required:"true"`
	StopHeadsign      String
	ArrivalTime       WideTime
	DepartureTime     WideTime
	PickupType        Int
	DropOffType       Int
	ContinuousPickup  Int
	ContinuousDropOff Int
	ShapeDistTraveled Float
	Timepoint         Int
	Interpolated      Int `csv:"-"` // interpolated times: 0 for provided, 1 interpolated // TODO: 1 for shape, 2 for straight-line
	FeedVersionID     int `csv:"-"`
	extra             []string
	loadErrors        []error
	loadWarnings      []error
}

// SetFeedVersionID sets the Entity's FeedVersionID.
func (ent *StopTime) SetFeedVersionID(fvid int) {
	ent.FeedVersionID = fvid
}

// AddError adds a loading error to the entity, e.g. from a CSV parse failure
func (ent *StopTime) AddError(err error) {
	ent.loadErrors = append(ent.loadErrors, err)
}

// AddWarning .
func (ent *StopTime) AddWarning(err error) {
	ent.loadWarnings = append(ent.loadErrors, err)
}

// Extra provides any additional fields that were present.
func (ent *StopTime) Extra() map[string]string {
	ret := map[string]string{}
	for i := 0; i < len(ent.extra); i += 2 {
		ret[ent.extra[i]] = ent.extra[i+1]
	}
	return ret
}

// SetExtra adds a string key, value pair to the entity's extra fields.
func (ent *StopTime) SetExtra(key string, value string) {
	ent.extra = append(ent.extra, key, value)
}

// EntityID returns nothing.
func (ent *StopTime) EntityID() string {
	return ""
}

// Errors for this Entity.
func (ent *StopTime) Errors() []error {
	// No reflection
	errs := []error{}
	errs = append(errs, ent.loadErrors...)
	errs = append(errs, tt.CheckPresent("trip_id", ent.TripID)...)
	errs = append(errs, tt.CheckPresent("stop_id", ent.StopID)...)
	errs = append(errs, tt.CheckPositiveInt("stop_sequence", ent.StopSequence)...)
	errs = append(errs, tt.CheckInsideRangeInt("pickup_type", ent.PickupType.Val, 0, 3)...)
	errs = append(errs, tt.CheckInsideRangeInt("drop_off_type", ent.DropOffType.Val, 0, 3)...)
	errs = append(errs, tt.CheckPositive("shape_dist_traveled", ent.ShapeDistTraveled.Val)...)
	errs = append(errs, tt.CheckInsideRangeInt("timepoint", ent.Timepoint.Val, -1, 1)...)
	errs = append(errs, tt.CheckInsideRangeInt("arrival_time", ent.ArrivalTime.Seconds, -1, 1<<31)...)
	errs = append(errs, tt.CheckInsideRangeInt("departure", ent.DepartureTime.Seconds, -1, 1<<31)...)
	errs = append(errs, tt.CheckInArrayInt("continuous_pickup", ent.ContinuousPickup.Val, 0, 1, 2, 3)...)
	errs = append(errs, tt.CheckInArrayInt("continuous_drop_off", ent.ContinuousDropOff.Val, 0, 1, 2, 3)...)
	// Other errors
	at, dt := ent.ArrivalTime.Seconds, ent.DepartureTime.Seconds
	if at != 0 && dt != 0 && at > dt {
		errs = append(errs, causes.NewInvalidFieldError("departure_time", "", fmt.Errorf("departure_time '%d' must come after arrival_time '%d'", dt, at)))
	}
	return errs
}

// Warnings for this Entity.
func (ent *StopTime) Warnings() []error {
	return ent.loadWarnings
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
		v = ent.StopHeadsign.String
	case "stop_id":
		v = ent.StopID
	case "arrival_time":
		v = ent.ArrivalTime.String()
	case "departure_time":
		v = ent.DepartureTime.String()
	case "stop_sequence":
		v = strconv.Itoa(ent.StopSequence)
	case "pickup_type":
		v = strconv.Itoa(int(ent.PickupType.Val))
	case "drop_off_type":
		v = strconv.Itoa(int(ent.DropOffType.Val))
	case "shape_dist_traveled":
		if ent.ShapeDistTraveled.Valid {
			v = fmt.Sprintf("%0.5f", ent.ShapeDistTraveled.Val)
		}
	case "timepoint":
		if ent.Timepoint.Valid {
			v = strconv.Itoa(int(ent.Timepoint.Val))
		}
	case "continuous_pickup":
		if ent.ContinuousPickup.Valid {
			v = strconv.Itoa(int(ent.ContinuousPickup.Val))
		}
	case "continuous_drop_off":
		if ent.ContinuousPickup.Valid {
			v = strconv.Itoa(int(ent.ContinuousDropOff.Val))
		}
	default:
		return v, fmt.Errorf("unknown key: %s", key)
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
		ent.StopHeadsign = NewString(hi)
	case "stop_id":
		ent.StopID = hi
	case "arrival_time":
		if hi == "" {
		} else if s, err := NewWideTime(hi); err != nil {
			perr = causes.NewFieldParseError("arrival_time", hi)
		} else {
			ent.ArrivalTime = s
		}
	case "departure_time":
		if hi == "" {
		} else if s, err := NewWideTime(hi); err != nil {
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
			ent.PickupType = Int{}
		} else if a, err := strconv.Atoi(hi); err != nil {
			perr = causes.NewFieldParseError("pickup_type", hi)
		} else {
			ent.PickupType = NewInt(a)
		}
	case "drop_off_type":
		if len(hi) == 0 {
			ent.DropOffType = Int{}
		} else if a, err := strconv.Atoi(hi); err != nil {
			perr = causes.NewFieldParseError("drop_off_type", hi)
		} else {
			ent.DropOffType = NewInt(a)
		}
	case "continuous_pickup":
		if len(hi) == 0 {
			ent.ContinuousPickup = Int{}
		} else if a, err := strconv.Atoi(hi); err != nil {
			perr = causes.NewFieldParseError("continuous_pickup", hi)
		} else {
			ent.ContinuousPickup = NewInt(a)
		}
	case "continuous_drop_off":
		if len(hi) == 0 {
			ent.ContinuousDropOff = Int{}
		} else if a, err := strconv.Atoi(hi); err != nil {
			perr = causes.NewFieldParseError("continuous_drop_off", hi)
		} else {
			ent.ContinuousDropOff = NewInt(a)
		}
	case "shape_dist_traveled":
		if len(hi) == 0 {
			ent.ShapeDistTraveled = Float{}
		} else if a, err := strconv.ParseFloat(hi, 64); err != nil {
			perr = causes.NewFieldParseError("shape_dist_traveled", hi)
		} else {
			ent.ShapeDistTraveled = NewFloat(a)
		}
	case "timepoint":
		// special use -1 for empty timepoint value
		if len(hi) == 0 {
			ent.Timepoint = Int{}
		} else if a, err := strconv.Atoi(hi); err != nil {
			perr = causes.NewFieldParseError("timepoint", hi)
		} else {
			ent.Timepoint = NewInt(a)
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
	if stoptimes[len(stoptimes)-1].ArrivalTime.Seconds <= 0 {
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
		if st.ArrivalTime.Seconds > 0 && st.ArrivalTime.Seconds < lastTime.Seconds {
			errs = append(errs, causes.NewSequenceError("arrival_time", st.ArrivalTime.String()))
		} else if st.DepartureTime.Seconds > 0 && st.DepartureTime.Seconds < st.ArrivalTime.Seconds {
			errs = append(errs, causes.NewSequenceError("departure_time", st.DepartureTime.String()))
		} else if st.DepartureTime.Seconds > 0 {
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
