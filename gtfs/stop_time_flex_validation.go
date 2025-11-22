package gtfs

import (
	"fmt"

	"github.com/interline-io/transitland-lib/causes"
)

// ConditionalErrors for StopTime - GTFS-Flex specific validation
func (ent *StopTime) conditionalErrorsFlex() (errs []error) {
	// 1. Validate pickup/drop_off window consistency
	hasStartWindow := ent.StartPickupDropOffWindow.Valid
	hasEndWindow := ent.EndPickupDropOffWindow.Valid

	if hasStartWindow && !hasEndWindow {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("end_pickup_drop_off_window"))
	}
	if hasEndWindow && !hasStartWindow {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("start_pickup_drop_off_window"))
	}

	// 2. If both windows are present, end must be >= start
	if hasStartWindow && hasEndWindow {
		if ent.EndPickupDropOffWindow.Val < ent.StartPickupDropOffWindow.Val {
			errs = append(errs, causes.NewInvalidFieldError(
				"end_pickup_drop_off_window",
				fmt.Sprintf("%d", ent.EndPickupDropOffWindow.Val),
				fmt.Errorf("must be greater than or equal to start_pickup_drop_off_window (%d)", ent.StartPickupDropOffWindow.Val),
			))
		}
	}

	// 3. Validate time windows vs fixed times
	// If using time windows, arrival_time and departure_time must be empty
	if hasStartWindow || hasEndWindow {
		if ent.ArrivalTime.Valid && ent.ArrivalTime.Int() > 0 {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError(
				"arrival_time",
				ent.ArrivalTime.String(),
				"arrival_time cannot be used with pickup/drop_off time windows",
			))
		}
		if ent.DepartureTime.Valid && ent.DepartureTime.Int() > 0 {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError(
				"departure_time",
				ent.DepartureTime.String(),
				"departure_time cannot be used with pickup/drop_off time windows",
			))
		}
	}

	// 4. Validate booking rules are only used with appropriate pickup/drop_off types
	// pickup_booking_rule_id can only be used when pickup_type = 2 (continuous pickup)
	// Note: Use IsPresent() to exclude empty strings, which are valid (meaning "no booking required")
	if ent.PickupBookingRuleID.IsPresent() {
		if !ent.PickupType.Valid || ent.PickupType.Val != 2 {
			errs = append(errs, causes.NewInvalidFieldError(
				"pickup_booking_rule_id",
				ent.PickupBookingRuleID.Val,
				fmt.Errorf("pickup_booking_rule_id requires pickup_type = 2 (on demand/continuous)"),
			))
		}
	}

	// drop_off_booking_rule_id can only be used when drop_off_type = 2
	// Note: Use IsPresent() to exclude empty strings, which are valid (meaning "no booking required")
	if ent.DropOffBookingRuleID.IsPresent() {
		if !ent.DropOffType.Valid || ent.DropOffType.Val != 2 {
			errs = append(errs, causes.NewInvalidFieldError(
				"drop_off_booking_rule_id",
				ent.DropOffBookingRuleID.Val,
				fmt.Errorf("drop_off_booking_rule_id requires drop_off_type = 2 (on demand/continuous)"),
			))
		}
	}

	// 5. Validate mean/safe duration factor/offset
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

	// safe_duration_factor should be >= mean_duration_factor if both present
	if ent.MeanDurationFactor.Valid && ent.SafeDurationFactor.Valid {
		if ent.SafeDurationFactor.Val < ent.MeanDurationFactor.Val {
			errs = append(errs, causes.NewInvalidFieldError(
				"safe_duration_factor",
				fmt.Sprintf("%f", ent.SafeDurationFactor.Val),
				fmt.Errorf("must be greater than or equal to mean_duration_factor (%f)", ent.MeanDurationFactor.Val),
			))
		}
	}

	return errs
}
