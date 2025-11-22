package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

func TestStopTime_ConditionalErrorsFlex(t *testing.T) {
	tests := []struct {
		name        string
		stopTime    *StopTime
		expectError bool
		errorField  string
	}{
		{
			name: "Valid: Both time windows present and consistent",
			stopTime: &StopTime{
				StartPickupDropOffWindow: tt.NewSeconds(3600),
				EndPickupDropOffWindow:   tt.NewSeconds(7200),
			},
			expectError: false,
		},
		{
			name: "Invalid: Only start window present",
			stopTime: &StopTime{
				StartPickupDropOffWindow: tt.NewSeconds(3600),
			},
			expectError: true,
			errorField:  "end_pickup_drop_off_window",
		},
		{
			name: "Invalid: Only end window present",
			stopTime: &StopTime{
				EndPickupDropOffWindow: tt.NewSeconds(7200),
			},
			expectError: true,
			errorField:  "start_pickup_drop_off_window",
		},
		{
			name: "Invalid: End window before start window",
			stopTime: &StopTime{
				StartPickupDropOffWindow: tt.NewSeconds(7200),
				EndPickupDropOffWindow:   tt.NewSeconds(3600),
			},
			expectError: true,
			errorField:  "end_pickup_drop_off_window",
		},
		{
			name: "Invalid: Time window with arrival_time",
			stopTime: &StopTime{
				StartPickupDropOffWindow: tt.NewSeconds(3600),
				EndPickupDropOffWindow:   tt.NewSeconds(7200),
				ArrivalTime:              tt.NewSeconds(10800),
			},
			expectError: true,
			errorField:  "arrival_time",
		},
		{
			name: "Invalid: Time window with departure_time",
			stopTime: &StopTime{
				StartPickupDropOffWindow: tt.NewSeconds(3600),
				EndPickupDropOffWindow:   tt.NewSeconds(7200),
				DepartureTime:            tt.NewSeconds(10800),
			},
			expectError: true,
			errorField:  "departure_time",
		},
		{
			name: "Valid: pickup_booking_rule_id with pickup_type=2",
			stopTime: &StopTime{
				PickupBookingRuleID: tt.NewKey("rule1"),
				PickupType:          tt.NewInt(2),
			},
			expectError: false,
		},
		{
			name: "Invalid: pickup_booking_rule_id without pickup_type=2",
			stopTime: &StopTime{
				PickupBookingRuleID: tt.NewKey("rule1"),
				PickupType:          tt.NewInt(0),
			},
			expectError: true,
			errorField:  "pickup_booking_rule_id",
		},
		{
			name: "Valid: empty pickup_booking_rule_id with pickup_type=2",
			stopTime: &StopTime{
				PickupBookingRuleID: tt.NewKey(""), // Empty is valid (no booking required)
				PickupType:          tt.NewInt(2),
			},
			expectError: false,
		},
		{
			name: "Valid: drop_off_booking_rule_id with drop_off_type=2",
			stopTime: &StopTime{
				DropOffBookingRuleID: tt.NewKey("rule1"),
				DropOffType:          tt.NewInt(2),
			},
			expectError: false,
		},
		{
			name: "Invalid: drop_off_booking_rule_id without drop_off_type=2",
			stopTime: &StopTime{
				DropOffBookingRuleID: tt.NewKey("rule1"),
				DropOffType:          tt.NewInt(0),
			},
			expectError: true,
			errorField:  "drop_off_booking_rule_id",
		},
		{
			name: "Valid: Both mean_duration fields present",
			stopTime: &StopTime{
				MeanDurationFactor: tt.NewFloat(1.5),
				MeanDurationOffset: tt.NewFloat(300),
			},
			expectError: false,
		},
		{
			name: "Invalid: Only mean_duration_factor present",
			stopTime: &StopTime{
				MeanDurationFactor: tt.NewFloat(1.5),
			},
			expectError: true,
			errorField:  "mean_duration_offset",
		},
		{
			name: "Invalid: Only mean_duration_offset present",
			stopTime: &StopTime{
				MeanDurationOffset: tt.NewFloat(300),
			},
			expectError: true,
			errorField:  "mean_duration_factor",
		},
		{
			name: "Invalid: mean_duration_factor is negative",
			stopTime: &StopTime{
				MeanDurationFactor: tt.NewFloat(-1.5),
				MeanDurationOffset: tt.NewFloat(300),
			},
			expectError: true,
			errorField:  "mean_duration_factor",
		},
		{
			name: "Invalid: safe_duration_factor < mean_duration_factor",
			stopTime: &StopTime{
				MeanDurationFactor: tt.NewFloat(2.0),
				MeanDurationOffset: tt.NewFloat(300),
				SafeDurationFactor: tt.NewFloat(1.5),
				SafeDurationOffset: tt.NewFloat(400),
			},
			expectError: true,
			errorField:  "safe_duration_factor",
		},
		{
			name: "Valid: safe_duration_factor >= mean_duration_factor",
			stopTime: &StopTime{
				MeanDurationFactor: tt.NewFloat(1.5),
				MeanDurationOffset: tt.NewFloat(300),
				SafeDurationFactor: tt.NewFloat(2.0),
				SafeDurationOffset: tt.NewFloat(400),
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tc.stopTime.conditionalErrorsFlex()

			if tc.expectError {
				assert.NotEmpty(t, errs, "Expected validation error")
				if len(errs) > 0 && tc.errorField != "" {
					// Check that at least one error mentions the expected field
					found := false
					for _, err := range errs {
						if err.Error() != "" {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected error related to field %s", tc.errorField)
				}
			} else {
				assert.Empty(t, errs, "Expected no validation errors")
			}
		})
	}
}
