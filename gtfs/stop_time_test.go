package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

func TestStopTime_Errors(t *testing.T) {
	tests := []struct {
		name           string
		stopTime       *StopTime
		expectedErrors []ExpectError
	}{
		{
			name: "Valid: Basic stop time",
			stopTime: &StopTime{
				TripID:        tt.NewString("trip1"),
				StopID:        tt.NewKey("stop1"),
				StopSequence:  tt.NewInt(1),
				ArrivalTime:   tt.NewSeconds(3600),
				DepartureTime: tt.NewSeconds(3630),
			},
			expectedErrors: nil,
		},
		{
			name: "Invalid: stop_sequence negative",
			stopTime: &StopTime{
				TripID:        tt.NewString("trip1"),
				StopID:        tt.NewKey("stop1"),
				StopSequence:  tt.NewInt(-1),
				ArrivalTime:   tt.NewSeconds(3600),
				DepartureTime: tt.NewSeconds(3630),
			},
			expectedErrors: PE("InvalidFieldError:stop_sequence"),
		},
		{
			name: "Invalid: pickup_type out of range",
			stopTime: &StopTime{
				TripID:        tt.NewString("trip1"),
				StopID:        tt.NewKey("stop1"),
				StopSequence:  tt.NewInt(1),
				ArrivalTime:   tt.NewSeconds(3600),
				DepartureTime: tt.NewSeconds(3630),
				PickupType:    tt.NewInt(4),
			},
			expectedErrors: PE("InvalidFieldError:pickup_type"),
		},
		{
			name: "Invalid: drop_off_type out of range",
			stopTime: &StopTime{
				TripID:        tt.NewString("trip1"),
				StopID:        tt.NewKey("stop1"),
				StopSequence:  tt.NewInt(1),
				ArrivalTime:   tt.NewSeconds(3600),
				DepartureTime: tt.NewSeconds(3630),
				DropOffType:   tt.NewInt(4),
			},
			expectedErrors: PE("InvalidFieldError:drop_off_type"),
		},
		{
			name: "Invalid: timepoint out of range",
			stopTime: &StopTime{
				TripID:        tt.NewString("trip1"),
				StopID:        tt.NewKey("stop1"),
				StopSequence:  tt.NewInt(1),
				ArrivalTime:   tt.NewSeconds(3600),
				DepartureTime: tt.NewSeconds(3630),
				Timepoint:     tt.NewInt(2),
			},
			expectedErrors: PE("InvalidFieldError:timepoint"),
		},
		{
			name: "Invalid: departure_time before arrival_time",
			stopTime: &StopTime{
				TripID:        tt.NewString("trip1"),
				StopID:        tt.NewKey("stop1"),
				StopSequence:  tt.NewInt(1),
				ArrivalTime:   tt.NewSeconds(3600),
				DepartureTime: tt.NewSeconds(1800),
			},
			expectedErrors: PE("InvalidFieldError:departure_time"),
		},
		{
			name: "Invalid: continuous_pickup out of range",
			stopTime: &StopTime{
				TripID:           tt.NewString("trip1"),
				StopID:           tt.NewKey("stop1"),
				StopSequence:     tt.NewInt(1),
				ArrivalTime:      tt.NewSeconds(3660),
				DepartureTime:    tt.NewSeconds(3690),
				ContinuousPickup: tt.NewInt(100),
			},
			expectedErrors: PE("InvalidFieldError:continuous_pickup"),
		},
		{
			name: "Invalid: continuous_drop_off out of range",
			stopTime: &StopTime{
				TripID:            tt.NewString("trip1"),
				StopID:            tt.NewKey("stop1"),
				StopSequence:      tt.NewInt(1),
				ArrivalTime:       tt.NewSeconds(3660),
				DepartureTime:     tt.NewSeconds(3690),
				ContinuousDropOff: tt.NewInt(100),
			},
			expectedErrors: PE("InvalidFieldError:continuous_drop_off"),
		},
		// ConditionalErrors - Location field mutual exclusion tests
		{
			name: "Valid: Only stop_id present",
			stopTime: &StopTime{
				TripID:       tt.NewString("trip1"),
				StopID:       tt.NewKey("stop1"),
				StopSequence: tt.NewInt(1),
			},
			expectedErrors: nil,
		},
		{
			name: "Valid: Only location_group_id present with time windows",
			stopTime: &StopTime{
				TripID:                   tt.NewString("trip1"),
				StopSequence:             tt.NewInt(1),
				LocationGroupID:          tt.NewKey("lg1"),
				StartPickupDropOffWindow: tt.NewSeconds(3600),
				EndPickupDropOffWindow:   tt.NewSeconds(7200),
			},
			expectedErrors: nil,
		},
		{
			name: "Valid: Only location_id present with time windows",
			stopTime: &StopTime{
				TripID:                   tt.NewString("trip1"),
				StopSequence:             tt.NewInt(1),
				LocationID:               tt.NewKey("loc1"),
				StartPickupDropOffWindow: tt.NewSeconds(3600),
				EndPickupDropOffWindow:   tt.NewSeconds(7200),
			},
			expectedErrors: nil,
		},
		{
			name: "Invalid: No location identifier",
			stopTime: &StopTime{
				TripID:       tt.NewString("trip1"),
				StopSequence: tt.NewInt(1),
			},
			expectedErrors: PE("ConditionallyRequiredFieldError:stop_id"),
		},
		{
			name: "Invalid: stop_id and location_group_id both present",
			stopTime: &StopTime{
				TripID:          tt.NewString("trip1"),
				StopSequence:    tt.NewInt(1),
				StopID:          tt.NewKey("stop1"),
				LocationGroupID: tt.NewKey("lg1"),
			},
			expectedErrors: PE(
				"ConditionallyForbiddenFieldError:location_group_id",
				"ConditionallyRequiredFieldError:start_pickup_drop_off_window",
				"ConditionallyRequiredFieldError:end_pickup_drop_off_window",
			),
		},
		{
			name: "Invalid: stop_id and location_id both present",
			stopTime: &StopTime{
				TripID:       tt.NewString("trip1"),
				StopSequence: tt.NewInt(1),
				StopID:       tt.NewKey("stop1"),
				LocationID:   tt.NewKey("loc1"),
			},
			expectedErrors: PE(
				"ConditionallyForbiddenFieldError:location_id",
				"ConditionallyRequiredFieldError:start_pickup_drop_off_window",
				"ConditionallyRequiredFieldError:end_pickup_drop_off_window",
			),
		},
		{
			name: "Invalid: location_group_id and location_id both present",
			stopTime: &StopTime{
				TripID:          tt.NewString("trip1"),
				StopSequence:    tt.NewInt(1),
				LocationGroupID: tt.NewKey("lg1"),
				LocationID:      tt.NewKey("loc1"),
			},
			expectedErrors: PE(
				"ConditionallyForbiddenFieldError:location_id",
				"ConditionallyRequiredFieldError:start_pickup_drop_off_window",
				"ConditionallyRequiredFieldError:end_pickup_drop_off_window",
			),
		},
		{
			name: "Invalid: location_group_id without time windows",
			stopTime: &StopTime{
				TripID:          tt.NewString("trip1"),
				StopSequence:    tt.NewInt(1),
				LocationGroupID: tt.NewKey("lg1"),
			},
			expectedErrors: PE(
				"ConditionallyRequiredFieldError:start_pickup_drop_off_window",
				"ConditionallyRequiredFieldError:end_pickup_drop_off_window",
			),
		},
		{
			name: "Invalid: location_id without time windows",
			stopTime: &StopTime{
				TripID:       tt.NewString("trip1"),
				StopSequence: tt.NewInt(1),
				LocationID:   tt.NewKey("loc1"),
			},
			expectedErrors: PE(
				"ConditionallyRequiredFieldError:start_pickup_drop_off_window",
				"ConditionallyRequiredFieldError:end_pickup_drop_off_window",
			),
		},
		{
			name: "Valid: Both time windows present and consistent",
			stopTime: &StopTime{
				TripID:                   tt.NewString("trip1"),
				StopSequence:             tt.NewInt(1),
				StopID:                   tt.NewKey("stop1"),
				StartPickupDropOffWindow: tt.NewSeconds(3600),
				EndPickupDropOffWindow:   tt.NewSeconds(7200),
			},
			expectedErrors: nil,
		},
		{
			name: "Invalid: Only start window present",
			stopTime: &StopTime{
				TripID:                   tt.NewString("trip1"),
				StopSequence:             tt.NewInt(1),
				StopID:                   tt.NewKey("stop1"),
				StartPickupDropOffWindow: tt.NewSeconds(3600),
			},
			expectedErrors: PE("ConditionallyRequiredFieldError:end_pickup_drop_off_window"),
		},
		{
			name: "Invalid: Only end window present",
			stopTime: &StopTime{
				TripID:                 tt.NewString("trip1"),
				StopSequence:           tt.NewInt(1),
				StopID:                 tt.NewKey("stop1"),
				EndPickupDropOffWindow: tt.NewSeconds(7200),
			},
			expectedErrors: PE("ConditionallyRequiredFieldError:start_pickup_drop_off_window"),
		},
		{
			name: "Invalid: End window before start window",
			stopTime: &StopTime{
				TripID:                   tt.NewString("trip1"),
				StopSequence:             tt.NewInt(1),
				StopID:                   tt.NewKey("stop1"),
				StartPickupDropOffWindow: tt.NewSeconds(7200),
				EndPickupDropOffWindow:   tt.NewSeconds(3600),
			},
			expectedErrors: PE("InvalidFieldError:end_pickup_drop_off_window"),
		},
		{
			name: "Invalid: Time window with arrival_time",
			stopTime: &StopTime{
				TripID:                   tt.NewString("trip1"),
				StopSequence:             tt.NewInt(1),
				StopID:                   tt.NewKey("stop1"),
				StartPickupDropOffWindow: tt.NewSeconds(3600),
				EndPickupDropOffWindow:   tt.NewSeconds(7200),
				ArrivalTime:              tt.NewSeconds(10800),
			},
			expectedErrors: PE("ConditionallyForbiddenFieldError:arrival_time"),
		},
		{
			name: "Invalid: Time window with departure_time",
			stopTime: &StopTime{
				TripID:                   tt.NewString("trip1"),
				StopSequence:             tt.NewInt(1),
				StopID:                   tt.NewKey("stop1"),
				StartPickupDropOffWindow: tt.NewSeconds(3600),
				EndPickupDropOffWindow:   tt.NewSeconds(7200),
				DepartureTime:            tt.NewSeconds(10800),
			},
			expectedErrors: PE("ConditionallyForbiddenFieldError:departure_time"),
		},
		// pickup_type and drop_off_type with time windows
		{
			name: "Invalid: pickup_type=0 with time windows",
			stopTime: &StopTime{
				TripID:                   tt.NewString("trip1"),
				StopSequence:             tt.NewInt(1),
				StopID:                   tt.NewKey("stop1"),
				StartPickupDropOffWindow: tt.NewSeconds(3600),
				EndPickupDropOffWindow:   tt.NewSeconds(7200),
				PickupType:               tt.NewInt(0),
			},
			expectedErrors: PE("ConditionallyForbiddenFieldError:pickup_type"),
		},
		{
			name: "Invalid: pickup_type=3 with time windows",
			stopTime: &StopTime{
				TripID:                   tt.NewString("trip1"),
				StopSequence:             tt.NewInt(1),
				StopID:                   tt.NewKey("stop1"),
				StartPickupDropOffWindow: tt.NewSeconds(3600),
				EndPickupDropOffWindow:   tt.NewSeconds(7200),
				PickupType:               tt.NewInt(3),
			},
			expectedErrors: PE("ConditionallyForbiddenFieldError:pickup_type"),
		},
		{
			name: "Valid: pickup_type=2 with time windows",
			stopTime: &StopTime{
				TripID:                   tt.NewString("trip1"),
				StopSequence:             tt.NewInt(1),
				StopID:                   tt.NewKey("stop1"),
				StartPickupDropOffWindow: tt.NewSeconds(3600),
				EndPickupDropOffWindow:   tt.NewSeconds(7200),
				PickupType:               tt.NewInt(2),
			},
			expectedErrors: nil,
		},
		{
			name: "Invalid: drop_off_type=0 with time windows",
			stopTime: &StopTime{
				TripID:                   tt.NewString("trip1"),
				StopSequence:             tt.NewInt(1),
				StopID:                   tt.NewKey("stop1"),
				StartPickupDropOffWindow: tt.NewSeconds(3600),
				EndPickupDropOffWindow:   tt.NewSeconds(7200),
				DropOffType:              tt.NewInt(0),
			},
			expectedErrors: PE("ConditionallyForbiddenFieldError:drop_off_type"),
		},
		{
			name: "Valid: drop_off_type=2 with time windows",
			stopTime: &StopTime{
				TripID:                   tt.NewString("trip1"),
				StopSequence:             tt.NewInt(1),
				StopID:                   tt.NewKey("stop1"),
				StartPickupDropOffWindow: tt.NewSeconds(3600),
				EndPickupDropOffWindow:   tt.NewSeconds(7200),
				DropOffType:              tt.NewInt(2),
			},
			expectedErrors: nil,
		},
		// continuous_pickup and continuous_drop_off with time windows
		{
			name: "Invalid: continuous_pickup=0 with time windows",
			stopTime: &StopTime{
				TripID:                   tt.NewString("trip1"),
				StopSequence:             tt.NewInt(1),
				StopID:                   tt.NewKey("stop1"),
				StartPickupDropOffWindow: tt.NewSeconds(3600),
				EndPickupDropOffWindow:   tt.NewSeconds(7200),
				ContinuousPickup:         tt.NewInt(0),
			},
			expectedErrors: PE("ConditionallyForbiddenFieldError:continuous_pickup"),
		},
		{
			name: "Invalid: continuous_pickup=2 with time windows",
			stopTime: &StopTime{
				TripID:                   tt.NewString("trip1"),
				StopSequence:             tt.NewInt(1),
				StopID:                   tt.NewKey("stop1"),
				StartPickupDropOffWindow: tt.NewSeconds(3600),
				EndPickupDropOffWindow:   tt.NewSeconds(7200),
				ContinuousPickup:         tt.NewInt(2),
			},
			expectedErrors: PE("ConditionallyForbiddenFieldError:continuous_pickup"),
		},
		{
			name: "Valid: continuous_pickup=1 with time windows",
			stopTime: &StopTime{
				TripID:                   tt.NewString("trip1"),
				StopSequence:             tt.NewInt(1),
				StopID:                   tt.NewKey("stop1"),
				StartPickupDropOffWindow: tt.NewSeconds(3600),
				EndPickupDropOffWindow:   tt.NewSeconds(7200),
				ContinuousPickup:         tt.NewInt(1),
			},
			expectedErrors: nil,
		},
		{
			name: "Invalid: continuous_drop_off=0 with time windows",
			stopTime: &StopTime{
				TripID:                   tt.NewString("trip1"),
				StopSequence:             tt.NewInt(1),
				StopID:                   tt.NewKey("stop1"),
				StartPickupDropOffWindow: tt.NewSeconds(3600),
				EndPickupDropOffWindow:   tt.NewSeconds(7200),
				ContinuousDropOff:        tt.NewInt(0),
			},
			expectedErrors: PE("ConditionallyForbiddenFieldError:continuous_drop_off"),
		},
		{
			name: "Valid: continuous_drop_off=1 with time windows",
			stopTime: &StopTime{
				TripID:                   tt.NewString("trip1"),
				StopSequence:             tt.NewInt(1),
				StopID:                   tt.NewKey("stop1"),
				StartPickupDropOffWindow: tt.NewSeconds(3600),
				EndPickupDropOffWindow:   tt.NewSeconds(7200),
				ContinuousDropOff:        tt.NewInt(1),
			},
			expectedErrors: nil,
		},
		{
			name: "Valid: pickup_booking_rule_id with pickup_type=2",
			stopTime: &StopTime{
				TripID:              tt.NewString("trip1"),
				StopSequence:        tt.NewInt(1),
				StopID:              tt.NewKey("stop1"),
				PickupBookingRuleID: tt.NewKey("rule1"),
				PickupType:          tt.NewInt(2),
			},
			expectedErrors: nil,
		},
		{
			name: "Valid: pickup_booking_rule_id with pickup_type=0",
			stopTime: &StopTime{
				TripID:              tt.NewString("trip1"),
				StopSequence:        tt.NewInt(1),
				StopID:              tt.NewKey("stop1"),
				PickupBookingRuleID: tt.NewKey("rule1"),
				PickupType:          tt.NewInt(0),
			},
			expectedErrors: nil,
		},
		{
			name: "Valid: drop_off_booking_rule_id with drop_off_type=2",
			stopTime: &StopTime{
				TripID:               tt.NewString("trip1"),
				StopSequence:         tt.NewInt(1),
				StopID:               tt.NewKey("stop1"),
				DropOffBookingRuleID: tt.NewKey("rule1"),
				DropOffType:          tt.NewInt(2),
			},
			expectedErrors: nil,
		},
		{
			name: "Valid: drop_off_booking_rule_id with drop_off_type=0",
			stopTime: &StopTime{
				TripID:               tt.NewString("trip1"),
				StopSequence:         tt.NewInt(1),
				StopID:               tt.NewKey("stop1"),
				DropOffBookingRuleID: tt.NewKey("rule1"),
				DropOffType:          tt.NewInt(0),
			},
			expectedErrors: nil,
		},
		{
			name: "Valid: Both mean_duration fields present",
			stopTime: &StopTime{
				TripID:             tt.NewString("trip1"),
				StopSequence:       tt.NewInt(1),
				StopID:             tt.NewKey("stop1"),
				MeanDurationFactor: tt.NewFloat(1.5),
				MeanDurationOffset: tt.NewFloat(300),
			},
			expectedErrors: nil,
		},
		{
			name: "Invalid: Only mean_duration_factor present",
			stopTime: &StopTime{
				TripID:             tt.NewString("trip1"),
				StopSequence:       tt.NewInt(1),
				StopID:             tt.NewKey("stop1"),
				MeanDurationFactor: tt.NewFloat(1.5),
			},
			expectedErrors: PE("ConditionallyRequiredFieldError:mean_duration_offset"),
		},
		{
			name: "Invalid: Only mean_duration_offset present",
			stopTime: &StopTime{
				TripID:             tt.NewString("trip1"),
				StopSequence:       tt.NewInt(1),
				StopID:             tt.NewKey("stop1"),
				MeanDurationOffset: tt.NewFloat(300),
			},
			expectedErrors: PE("ConditionallyRequiredFieldError:mean_duration_factor"),
		},
		{
			name: "Invalid: mean_duration_factor is negative",
			stopTime: &StopTime{
				TripID:             tt.NewString("trip1"),
				StopSequence:       tt.NewInt(1),
				StopID:             tt.NewKey("stop1"),
				MeanDurationFactor: tt.NewFloat(-1.5),
				MeanDurationOffset: tt.NewFloat(300),
			},
			expectedErrors: PE("InvalidFieldError:mean_duration_factor"),
		},
		{
			name: "Valid: safe_duration_factor with mean_duration_factor",
			stopTime: &StopTime{
				TripID:             tt.NewString("trip1"),
				StopSequence:       tt.NewInt(1),
				StopID:             tt.NewKey("stop1"),
				MeanDurationFactor: tt.NewFloat(1.5),
				MeanDurationOffset: tt.NewFloat(300),
				SafeDurationFactor: tt.NewFloat(2.0),
				SafeDurationOffset: tt.NewFloat(400),
			},
			expectedErrors: nil,
		},
		{
			name: "Invalid: Missing trip_id",
			stopTime: &StopTime{
				StopID:       tt.NewKey("stop1"),
				StopSequence: tt.NewInt(1),
			},
			expectedErrors: PE("RequiredFieldError:trip_id"),
		},
		{
			name: "Invalid: shape_dist_traveled negative",
			stopTime: &StopTime{
				TripID:            tt.NewString("trip1"),
				StopID:            tt.NewKey("stop1"),
				StopSequence:      tt.NewInt(1),
				ShapeDistTraveled: tt.NewFloat(-1.0),
			},
			expectedErrors: PE("InvalidFieldError:shape_dist_traveled"),
		},
		{
			name: "Invalid: Only safe_duration_factor present",
			stopTime: &StopTime{
				TripID:             tt.NewString("trip1"),
				StopSequence:       tt.NewInt(1),
				StopID:             tt.NewKey("stop1"),
				SafeDurationFactor: tt.NewFloat(1.5),
			},
			expectedErrors: PE("ConditionallyRequiredFieldError:safe_duration_offset"),
		},
		{
			name: "Invalid: Only safe_duration_offset present",
			stopTime: &StopTime{
				TripID:             tt.NewString("trip1"),
				StopSequence:       tt.NewInt(1),
				StopID:             tt.NewKey("stop1"),
				SafeDurationOffset: tt.NewFloat(300),
			},
			expectedErrors: PE("ConditionallyRequiredFieldError:safe_duration_factor"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.stopTime)
			CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}
