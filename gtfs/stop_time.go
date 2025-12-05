package gtfs

import (
	"fmt"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// StopTime stop_times.txt
type StopTime struct {
	TripID            tt.String `csv:",required" target:"trips.txt"`
	StopID            tt.Key    `target:"stops.txt"`
	LocationGroupID   tt.Key    `target:"location_groups.txt"`
	LocationID        tt.Key    `target:"locations.txt"`
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
	// Check which location identifier is used
	hasStopID := ent.StopID.IsPresent()
	hasLocationGroupID := ent.LocationGroupID.IsPresent()
	hasLocationID := ent.LocationID.IsPresent()
	hasTimeWindow := ent.StartPickupDropOffWindow.Valid || ent.EndPickupDropOffWindow.Valid

	// 1. Mutual exclusion: stop_id, location_id, location_group_id
	// Exactly one of these must be defined
	if !hasStopID && !hasLocationGroupID && !hasLocationID {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("stop_id"))
	}

	if hasStopID && hasLocationGroupID {
		errs = append(errs, causes.NewConditionallyForbiddenFieldError(
			"location_group_id",
			ent.LocationGroupID.Val,
			"location_group_id is forbidden when stop_id is defined",
		))
	}
	if hasStopID && hasLocationID {
		errs = append(errs, causes.NewConditionallyForbiddenFieldError(
			"location_id",
			ent.LocationID.Val,
			"location_id is forbidden when stop_id is defined",
		))
	}
	if hasLocationGroupID && hasLocationID {
		errs = append(errs, causes.NewConditionallyForbiddenFieldError(
			"location_id",
			ent.LocationID.Val,
			"location_id is forbidden when location_group_id is defined",
		))
	}

	// 2. Time windows required if location_group_id or location_id is defined
	if hasLocationGroupID || hasLocationID {
		if !ent.StartPickupDropOffWindow.Valid {
			errs = append(errs, causes.NewConditionallyRequiredFieldError("start_pickup_drop_off_window"))
		}
		if !ent.EndPickupDropOffWindow.Valid {
			errs = append(errs, causes.NewConditionallyRequiredFieldError("end_pickup_drop_off_window"))
		}
	}

	// 3. Time windows consistency
	hasStartWindow := ent.StartPickupDropOffWindow.Valid
	hasEndWindow := ent.EndPickupDropOffWindow.Valid

	if hasStartWindow && !hasEndWindow {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("end_pickup_drop_off_window"))
	}
	if hasEndWindow && !hasStartWindow {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("start_pickup_drop_off_window"))
	}

	// 4. If both windows are present, end must be >= start
	if hasStartWindow && hasEndWindow {
		if ent.EndPickupDropOffWindow.Val < ent.StartPickupDropOffWindow.Val {
			errs = append(errs, causes.NewInvalidFieldError(
				"end_pickup_drop_off_window",
				fmt.Sprintf("%d", ent.EndPickupDropOffWindow.Val),
				fmt.Errorf("must be greater than or equal to start_pickup_drop_off_window (%d)", ent.StartPickupDropOffWindow.Val),
			))
		}
	}

	// 5. Time windows vs fixed times
	// If using time windows, arrival_time and departure_time are forbidden
	if hasTimeWindow {
		if ent.ArrivalTime.Valid && ent.ArrivalTime.Int() > 0 {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError(
				"arrival_time",
				ent.ArrivalTime.String(),
				"arrival_time is forbidden when start_pickup_drop_off_window or end_pickup_drop_off_window are defined",
			))
		}
		if ent.DepartureTime.Valid && ent.DepartureTime.Int() > 0 {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError(
				"departure_time",
				ent.DepartureTime.String(),
				"departure_time is forbidden when start_pickup_drop_off_window or end_pickup_drop_off_window are defined",
			))
		}
	}

	// 6. pickup_type restrictions with time windows
	// pickup_type=0 (regularly scheduled) is forbidden if time windows are defined
	// pickup_type=3 (coordinate with driver) is forbidden if time windows are defined
	if hasTimeWindow && ent.PickupType.Valid {
		if ent.PickupType.Val == 0 {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError(
				"pickup_type",
				fmt.Sprintf("%d", ent.PickupType.Val),
				"pickup_type=0 (regularly scheduled) is forbidden when time windows are defined",
			))
		}
		if ent.PickupType.Val == 3 {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError(
				"pickup_type",
				fmt.Sprintf("%d", ent.PickupType.Val),
				"pickup_type=3 (coordinate with driver) is forbidden when time windows are defined",
			))
		}
	}

	// 7. drop_off_type restrictions with time windows
	// drop_off_type=0 (regularly scheduled) is forbidden if time windows are defined
	if hasTimeWindow && ent.DropOffType.Valid && ent.DropOffType.Val == 0 {
		errs = append(errs, causes.NewConditionallyForbiddenFieldError(
			"drop_off_type",
			fmt.Sprintf("%d", ent.DropOffType.Val),
			"drop_off_type=0 (regularly scheduled) is forbidden when time windows are defined",
		))
	}

	// 8. continuous_pickup restrictions with time windows
	// Any value other than 1 or empty is forbidden if time windows are defined
	if hasTimeWindow && ent.ContinuousPickup.Valid {
		if ent.ContinuousPickup.Val != 1 {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError(
				"continuous_pickup",
				fmt.Sprintf("%d", ent.ContinuousPickup.Val),
				"continuous_pickup must be 1 (no continuous pickup) or empty when time windows are defined",
			))
		}
	}

	// 9. continuous_drop_off restrictions with time windows
	// Any value other than 1 or empty is forbidden if time windows are defined
	if hasTimeWindow && ent.ContinuousDropOff.Valid {
		if ent.ContinuousDropOff.Val != 1 {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError(
				"continuous_drop_off",
				fmt.Sprintf("%d", ent.ContinuousDropOff.Val),
				"continuous_drop_off must be 1 (no continuous drop off) or empty when time windows are defined",
			))
		}
	}

	// 10. Validate mean/safe duration factor/offset
	// mean_duration_factor and mean_duration_offset work together
	if ent.MeanDurationFactor.Valid && !ent.MeanDurationOffset.Valid {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("mean_duration_offset"))
	}
	if ent.MeanDurationOffset.Valid && !ent.MeanDurationFactor.Valid {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("mean_duration_factor"))
	}

	// safe_duration_factor and safe_duration_offset work together
	if ent.SafeDurationFactor.Valid && !ent.SafeDurationOffset.Valid {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("safe_duration_offset"))
	}
	if ent.SafeDurationOffset.Valid && !ent.SafeDurationFactor.Valid {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("safe_duration_factor"))
	}

	// mean_duration_factor should be positive
	if ent.MeanDurationFactor.Valid && ent.MeanDurationFactor.Val <= 0 {
		errs = append(errs, causes.NewInvalidFieldError(
			"mean_duration_factor",
			fmt.Sprintf("%f", ent.MeanDurationFactor.Val),
			fmt.Errorf("must be positive"),
		))
	}

	return errs
}

// UpdateKeys updates Entity references.
func (ent *StopTime) UpdateKeys(emap *tt.EntityMap) error {
	// Don't use reflection based path
	return tt.FirstError(
		tt.TrySetField(emap.UpdateKey(&ent.TripID, "trips.txt"), "trip_id"),
		tt.TrySetField(emap.UpdateKey(&ent.StopID, "stops.txt"), "stop_id"),
		tt.TrySetField(emap.UpdateKey(&ent.LocationGroupID, "location_groups.txt"), "location_group_id"),
		tt.TrySetField(emap.UpdateKey(&ent.LocationID, "locations.geojson"), "location_id"),
		tt.TrySetField(emap.UpdateKey(&ent.PickupBookingRuleID, "booking_rules.txt"), "pickup_booking_rule_id"),
		tt.TrySetField(emap.UpdateKey(&ent.DropOffBookingRuleID, "booking_rules.txt"), "drop_off_booking_rule_id"),
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
	case "location_group_id":
		v = ent.LocationGroupID.Val
	case "location_id":
		v = ent.LocationID.Val
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
	case "location_group_id":
		ent.LocationGroupID.Set(hi)
	case "location_id":
		ent.LocationID.Set(hi)
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
