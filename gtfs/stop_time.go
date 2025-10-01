package gtfs

import (
	"fmt"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// StopTime stop_times.txt
type StopTime struct {
	TripID            tt.String `csv:",required" target:"trips.txt"`
	StopID            tt.String `csv:",required" target:"stops.txt"`
	StopSequence      tt.Int    `csv:",required"`
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

// Filename stop_times.txt
func (ent *StopTime) Filename() string {
	return "stop_times.txt"
}

// TableName gtfs_stop_times
func (ent *StopTime) TableName() string {
	return "gtfs_stop_times"
}

// Errors for this Entity.
func (ent *StopTime) Errors() []error {
	// Don't use reflection based path
	errs := []error{}
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

// UpdateKeys updates Entity references.
func (ent *StopTime) UpdateKeys(emap *tt.EntityMap) error {
	// Don't use reflection based path
	return tt.FirstError(
		tt.TrySetField(emap.UpdateKey(&ent.TripID, "trips.txt"), "trip_id"),
		tt.TrySetField(emap.UpdateKey(&ent.StopID, "stops.txt"), "stop_id"),
	)
}

// GetString returns the string representation of an field.
func (ent *StopTime) GetString(key string) (string, error) {
	// Don't use reflection based path
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
	// Don't use reflection based path
	var perr error
	hi := value
	switch key {
	case "trip_id":
		ent.TripID.Set(hi)
	case "stop_headsign":
		ent.StopHeadsign.Set(hi)
	case "stop_id":
		ent.StopID.Set(hi)
	case "arrival_time":
		perr = ent.ArrivalTime.Scan(hi)
	case "departure_time":
		perr = ent.DepartureTime.Scan(hi)
	case "stop_sequence":
		perr = ent.StopSequence.Scan(hi)
	case "pickup_type":
		perr = ent.PickupType.Scan(hi)
	case "drop_off_type":
		perr = ent.DropOffType.Scan(hi)
	case "continuous_pickup":
		perr = ent.ContinuousPickup.Scan(hi)
	case "continuous_drop_off":
		perr = ent.ContinuousDropOff.Scan(hi)
	case "shape_dist_traveled":
		perr = ent.ShapeDistTraveled.Scan(hi)
	case "timepoint":
		perr = ent.Timepoint.Scan(hi)
	default:
		ent.SetExtra(key, hi)
	}
	return perr
}
