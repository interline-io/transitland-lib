package gtfs

import (
	"fmt"
	"strconv"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// StopTime stop_times.txt
type StopTime struct {
	TripID            tt.String
	StopID            tt.String
	StopSequence      tt.Int
	StopHeadsign      tt.String
	ArrivalTime       tt.Seconds
	DepartureTime     tt.Seconds
	PickupType        tt.Int
	DropOffType       tt.Int
	ContinuousPickup  tt.Int
	ContinuousDropOff tt.Int
	ShapeDistTraveled tt.Float
	Timepoint         tt.Int
	Interpolated      tt.Int `csv:"-"` // interpolated times: 0 for provided, 1 interpolated // TODO: 1 for shape, 2 for straight-line
	tt.MinEntity
	tt.ErrorEntity
	tt.ExtraEntity
	tt.FeedVersionEntity
}

// Errors for this Entity.
func (ent *StopTime) Errors() []error {
	// No reflection
	errs := []error{}
	errs = append(errs, ent.ErrorEntity.Errors()...)
	errs = append(errs, tt.CheckPresent("trip_id", ent.TripID.Val)...)
	errs = append(errs, tt.CheckPresent("stop_id", ent.StopID.Val)...)
	errs = append(errs, tt.CheckPositiveInt("stop_sequence", ent.StopSequence.Val)...)
	errs = append(errs, tt.CheckInsideRangeInt("pickup_type", ent.PickupType.Val, 0, 3)...)
	errs = append(errs, tt.CheckInsideRangeInt("drop_off_type", ent.DropOffType.Val, 0, 3)...)
	errs = append(errs, tt.CheckPositive("shape_dist_traveled", ent.ShapeDistTraveled.Val)...)
	errs = append(errs, tt.CheckInsideRangeInt("timepoint", ent.Timepoint.Val, -1, 1)...)
	errs = append(errs, tt.CheckInsideRangeInt("arrival_time", ent.ArrivalTime.Int(), -1, 1<<31)...)
	errs = append(errs, tt.CheckInsideRangeInt("departure", ent.DepartureTime.Int(), -1, 1<<31)...)
	errs = append(errs, tt.CheckInArrayInt("continuous_pickup", ent.ContinuousPickup.Val, 0, 1, 2, 3)...)
	errs = append(errs, tt.CheckInArrayInt("continuous_drop_off", ent.ContinuousDropOff.Val, 0, 1, 2, 3)...)
	// Other errors
	at, dt := ent.ArrivalTime.Int(), ent.DepartureTime.Int()
	if at != 0 && dt != 0 && at > dt {
		errs = append(errs, causes.NewInvalidFieldError("departure_time", ent.DepartureTime.String(), fmt.Errorf("departure_time '%d' must come after arrival_time '%d'", dt, at)))
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
	if tripID, ok := emap.GetEntity(&Trip{TripID: ent.TripID.Val}); ok {
		ent.TripID.Set(tripID)
	} else {
		return causes.NewInvalidReferenceError("trip_id", ent.TripID.Val)
	}
	if stopID, ok := emap.GetEntity(&Stop{StopID: ent.StopID.Val}); ok {
		ent.StopID.Set(stopID)
	} else {
		return causes.NewInvalidReferenceError("stop_id", ent.StopID.Val)
	}
	return nil
}

// GetString returns the string representation of an field.
func (ent *StopTime) GetString(key string) (string, error) {
	v := ""
	switch key {
	case "trip_id":
		v = ent.TripID.Val
	case "stop_headsign":
		v = ent.StopHeadsign.Val
	case "stop_id":
		v = ent.StopID.Val
	case "arrival_time":
		v = ent.ArrivalTime.String()
	case "departure_time":
		v = ent.DepartureTime.String()
	case "stop_sequence":
		v = ent.StopSequence.String()
	case "pickup_type":
		v = ent.PickupType.String()
	case "drop_off_type":
		v = ent.DropOffType.String()
	case "shape_dist_traveled":
		if ent.ShapeDistTraveled.Valid {
			v = fmt.Sprintf("%0.5f", ent.ShapeDistTraveled.Val)
		}
	case "timepoint":
		v = ent.Timepoint.String()
	case "continuous_pickup":
		v = ent.ContinuousPickup.String()
	case "continuous_drop_off":
		v = ent.ContinuousDropOff.String()
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
		ent.TripID.Set(hi)
	case "stop_headsign":
		ent.StopHeadsign = tt.NewString(hi)
	case "stop_id":
		ent.StopID.Set(hi)
	case "arrival_time":
		if hi == "" {
		} else if s, err := tt.NewSecondsFromString(hi); err != nil {
			perr = causes.NewFieldParseError("arrival_time", hi)
		} else {
			ent.ArrivalTime = s
		}
	case "departure_time":
		if hi == "" {
		} else if s, err := tt.NewSecondsFromString(hi); err != nil {
			perr = causes.NewFieldParseError("departure_time", hi)
		} else {
			ent.DepartureTime = s
		}
	case "stop_sequence":
		if a, err := strconv.Atoi(hi); err != nil {
			perr = causes.NewFieldParseError("stop_sequence", hi)
		} else {
			ent.StopSequence.Set(int64(a))
		}
	case "pickup_type":
		if hi == "" {
			ent.PickupType = tt.Int{}
		} else if a, err := strconv.Atoi(hi); err != nil {
			perr = causes.NewFieldParseError("pickup_type", hi)
		} else {
			ent.PickupType = tt.NewInt(a)
		}
	case "drop_off_type":
		if hi == "" {
			ent.DropOffType = tt.Int{}
		} else if a, err := strconv.Atoi(hi); err != nil {
			perr = causes.NewFieldParseError("drop_off_type", hi)
		} else {
			ent.DropOffType = tt.NewInt(a)
		}
	case "continuous_pickup":
		if hi == "" {
			ent.ContinuousPickup = tt.Int{}
		} else if a, err := strconv.Atoi(hi); err != nil {
			perr = causes.NewFieldParseError("continuous_pickup", hi)
		} else {
			ent.ContinuousPickup = tt.NewInt(a)
		}
	case "continuous_drop_off":
		if hi == "" {
			ent.ContinuousDropOff = tt.Int{}
		} else if a, err := strconv.Atoi(hi); err != nil {
			perr = causes.NewFieldParseError("continuous_drop_off", hi)
		} else {
			ent.ContinuousDropOff = tt.NewInt(a)
		}
	case "shape_dist_traveled":
		if hi == "" {
			ent.ShapeDistTraveled = tt.Float{}
		} else if a, err := strconv.ParseFloat(hi, 64); err != nil {
			perr = causes.NewFieldParseError("shape_dist_traveled", hi)
		} else {
			ent.ShapeDistTraveled = tt.NewFloat(a)
		}
	case "timepoint":
		if hi == "" {
			ent.Timepoint = tt.Int{}
		} else if a, err := strconv.Atoi(hi); err != nil {
			perr = causes.NewFieldParseError("timepoint", hi)
		} else {
			ent.Timepoint = tt.NewInt(a)
		}
	default:
		ent.SetExtra(key, hi)
	}
	return perr
}
