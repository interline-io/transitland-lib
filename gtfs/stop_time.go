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
	// GTFS-Flex fields (officially adopted)
	StartPickupDropOffWindow tt.Seconds
	EndPickupDropOffWindow   tt.Seconds
	PickupBookingRuleID      tt.Key `target:"booking_rules.txt"`
	DropOffBookingRuleID     tt.Key `target:"booking_rules.txt"`
	// GTFS-Flex proposed fields (not yet formally adopted, may change)
	// See: https://github.com/MobilityData/gtfs-flex/blob/master/spec/reference.md
	MeanDurationFactor tt.Float // proposed gtfs-flex
	MeanDurationOffset tt.Float // proposed gtfs-flex
	SafeDurationFactor tt.Float // proposed gtfs-flex
	SafeDurationOffset tt.Float // proposed gtfs-flex
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

// ConditionalErrors for StopTime - includes GTFS-Flex validation
func (ent *StopTime) ConditionalErrors() (errs []error) {
	errs = append(errs, ent.conditionalErrorsFlex()...)
	return errs
}

// UpdateKeys updates Entity references.
func (ent *StopTime) UpdateKeys(emap *tt.EntityMap) error {
	// Don't use reflection based path
	return tt.FirstError(
		tt.TrySetField(emap.UpdateKey(&ent.TripID, "trips.txt"), "trip_id"),
		tt.TrySetField(emap.UpdateKey(&ent.StopID, "stops.txt"), "stop_id"),
		tt.TrySetField(emap.UpdateKey(&ent.PickupBookingRuleID, "booking_rules.txt"), "booking_rule_id"),
		tt.TrySetField(emap.UpdateKey(&ent.DropOffBookingRuleID, "booking_rules.txt"), "booking_rule_id"),
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
	case "start_pickup_drop_off_window":
		v = ent.StartPickupDropOffWindow.String()
	case "end_pickup_drop_off_window":
		v = ent.EndPickupDropOffWindow.String()
	case "pickup_booking_rule_id":
		v = ent.PickupBookingRuleID.Val
	case "drop_off_booking_rule_id":
		v = ent.DropOffBookingRuleID.Val
	case "mean_duration_factor":
		if ent.MeanDurationFactor.Valid {
			v = fmt.Sprintf("%0.5f", ent.MeanDurationFactor.Val)
		}
	case "mean_duration_offset":
		if ent.MeanDurationOffset.Valid {
			v = fmt.Sprintf("%0.5f", ent.MeanDurationOffset.Val)
		}
	case "safe_duration_factor":
		if ent.SafeDurationFactor.Valid {
			v = fmt.Sprintf("%0.5f", ent.SafeDurationFactor.Val)
		}
	case "safe_duration_offset":
		if ent.SafeDurationOffset.Valid {
			v = fmt.Sprintf("%0.5f", ent.SafeDurationOffset.Val)
		}
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
		if err := ent.ArrivalTime.Scan(hi); err != nil {
			perr = causes.NewFieldParseError("arrival_time", hi)
		}
	case "departure_time":
		if err := ent.DepartureTime.Scan(hi); err != nil {
			perr = causes.NewFieldParseError("departure_time", hi)
		}
	case "stop_sequence":
		if err := ent.StopSequence.Scan(hi); err != nil {
			perr = causes.NewFieldParseError("stop_sequence", hi)
		}
	case "pickup_type":
		if err := ent.PickupType.Scan(hi); err != nil {
			perr = causes.NewFieldParseError("pickup_type", hi)
		}
	case "drop_off_type":
		if err := ent.DropOffType.Scan(hi); err != nil {
			perr = causes.NewFieldParseError("drop_off_type", hi)
		}
	case "continuous_pickup":
		if err := ent.ContinuousPickup.Scan(hi); err != nil {
			perr = causes.NewFieldParseError("continuous_pickup", hi)
		}
	case "continuous_drop_off":
		if err := ent.ContinuousDropOff.Scan(hi); err != nil {
			perr = causes.NewFieldParseError("continuous_drop_off", hi)
		}
	case "shape_dist_traveled":
		if err := ent.ShapeDistTraveled.Scan(hi); err != nil {
			perr = causes.NewFieldParseError("shape_dist_traveled", hi)
		}
	case "timepoint":
		if err := ent.Timepoint.Scan(hi); err != nil {
			perr = causes.NewFieldParseError("timepoint", hi)
		}
	case "start_pickup_drop_off_window":
		if err := ent.StartPickupDropOffWindow.Scan(hi); err != nil {
			perr = causes.NewFieldParseError("start_pickup_drop_off_window", hi)
		}
	case "end_pickup_drop_off_window":
		if err := ent.EndPickupDropOffWindow.Scan(hi); err != nil {
			perr = causes.NewFieldParseError("end_pickup_drop_off_window", hi)
		}
	case "pickup_booking_rule_id":
		ent.PickupBookingRuleID.Set(hi)
	case "drop_off_booking_rule_id":
		ent.DropOffBookingRuleID.Set(hi)
	case "mean_duration_factor":
		if err := ent.MeanDurationFactor.Scan(hi); err != nil {
			perr = causes.NewFieldParseError("mean_duration_factor", hi)
		}
	case "mean_duration_offset":
		if err := ent.MeanDurationOffset.Scan(hi); err != nil {
			perr = causes.NewFieldParseError("mean_duration_offset", hi)
		}
	case "safe_duration_factor":
		if err := ent.SafeDurationFactor.Scan(hi); err != nil {
			perr = causes.NewFieldParseError("safe_duration_factor", hi)
		}
	case "safe_duration_offset":
		if err := ent.SafeDurationOffset.Scan(hi); err != nil {
			perr = causes.NewFieldParseError("safe_duration_offset", hi)
		}
	default:
		ent.SetExtra(key, hi)
	}
	return perr
}
