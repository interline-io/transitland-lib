package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

func TestBookingRule_ConditionalErrors(t *testing.T) {
	tests := []struct {
		name           string
		bookingRule    *BookingRule
		expectedErrors []ExpectError
	}{
		// ===== BOOKING_TYPE=0 TESTS =====
		{
			name: "Valid: booking_type=0 with minimal fields",
			bookingRule: &BookingRule{
				BookingRuleID: tt.NewString("rule1"),
				BookingType:   tt.NewInt(0),
			},
			expectedErrors: nil,
		},
		{
			name: "Invalid: booking_type=0 with prior_notice_duration_min",
			bookingRule: &BookingRule{
				BookingRuleID:          tt.NewString("rule1_invalid"),
				BookingType:            tt.NewInt(0),
				PriorNoticeDurationMin: tt.NewInt(15),
			},
			expectedErrors: ParseExpectErrors("ConditionallyForbiddenFieldError:prior_notice_duration_min"),
		},
		{
			name: "Invalid: booking_type=0 with prior_notice_duration_max",
			bookingRule: &BookingRule{
				BookingRuleID:          tt.NewString("rule2"),
				BookingType:            tt.NewInt(0),
				PriorNoticeDurationMax: tt.NewInt(60),
			},
			expectedErrors: ParseExpectErrors("ConditionallyForbiddenFieldError:prior_notice_duration_max"),
		},
		{
			name: "Invalid: booking_type=0 with prior_notice_start_day",
			bookingRule: &BookingRule{
				BookingRuleID:        tt.NewString("rule3"),
				BookingType:          tt.NewInt(0),
				PriorNoticeStartDay:  tt.NewInt(1),
				PriorNoticeStartTime: tt.NewSeconds(3600),
			},
			expectedErrors: ParseExpectErrors("ConditionallyForbiddenFieldError:prior_notice_start_day"),
		},
		{
			name: "Invalid: booking_type=0 with both forbidden fields",
			bookingRule: &BookingRule{
				BookingRuleID:          tt.NewString("rule4"),
				BookingType:            tt.NewInt(0),
				PriorNoticeDurationMax: tt.NewInt(60),
				PriorNoticeStartDay:    tt.NewInt(1),
				PriorNoticeStartTime:   tt.NewSeconds(3600),
			},
			expectedErrors: ParseExpectErrors(
				"ConditionallyForbiddenFieldError:prior_notice_duration_max",
				"ConditionallyForbiddenFieldError:prior_notice_start_day",
			),
		},
		{
			name: "Invalid: booking_type=0 with prior_notice_service_id",
			bookingRule: &BookingRule{
				BookingRuleID:        tt.NewString("rule5"),
				BookingType:          tt.NewInt(0),
				PriorNoticeServiceID: tt.NewKey("service1"),
			},
			expectedErrors: ParseExpectErrors("ConditionallyForbiddenFieldError:prior_notice_service_id"),
		},

		// ===== BOOKING_TYPE=1 TESTS =====
		{
			name: "Invalid: booking_type=1 without prior_notice_duration_min",
			bookingRule: &BookingRule{
				BookingRuleID: tt.NewString("rule6"),
				BookingType:   tt.NewInt(1),
			},
			expectedErrors: ParseExpectErrors("ConditionallyRequiredFieldError:prior_notice_duration_min"),
		},
		{
			name: "Valid: booking_type=1 with required fields",
			bookingRule: &BookingRule{
				BookingRuleID:          tt.NewString("rule7"),
				BookingType:            tt.NewInt(1),
				PriorNoticeDurationMin: tt.NewInt(30),
			},
			expectedErrors: nil,
		},
		{
			name: "Valid: booking_type=1 with optional prior_notice_duration_max",
			bookingRule: &BookingRule{
				BookingRuleID:          tt.NewString("rule8"),
				BookingType:            tt.NewInt(1),
				PriorNoticeDurationMin: tt.NewInt(30),
				PriorNoticeDurationMax: tt.NewInt(120),
			},
			expectedErrors: nil,
		},
		{
			name: "Invalid: booking_type=1 with prior_notice_duration_max and prior_notice_start_day",
			bookingRule: &BookingRule{
				BookingRuleID:          tt.NewString("rule9"),
				BookingType:            tt.NewInt(1),
				PriorNoticeDurationMin: tt.NewInt(30),
				PriorNoticeDurationMax: tt.NewInt(120),
				PriorNoticeStartDay:    tt.NewInt(1),
				PriorNoticeStartTime:   tt.NewSeconds(3600),
			},
			expectedErrors: ParseExpectErrors("ConditionallyForbiddenFieldError:prior_notice_start_day"),
		},
		{
			name: "Valid: booking_type=1 with prior_notice_start_day without prior_notice_duration_max",
			bookingRule: &BookingRule{
				BookingRuleID:          tt.NewString("rule10"),
				BookingType:            tt.NewInt(1),
				PriorNoticeDurationMin: tt.NewInt(30),
				PriorNoticeStartDay:    tt.NewInt(1),
				PriorNoticeStartTime:   tt.NewSeconds(3600),
			},
			expectedErrors: nil,
		},
		{
			name: "Invalid: booking_type=1 with prior_notice_service_id",
			bookingRule: &BookingRule{
				BookingRuleID:          tt.NewString("rule11"),
				BookingType:            tt.NewInt(1),
				PriorNoticeDurationMin: tt.NewInt(30),
				PriorNoticeServiceID:   tt.NewKey("service1"),
			},
			expectedErrors: ParseExpectErrors("ConditionallyForbiddenFieldError:prior_notice_service_id"),
		},
		{
			name: "Invalid: booking_type=1 with prior_notice_last_day",
			bookingRule: &BookingRule{
				BookingRuleID:          tt.NewString("rule11_last_day"),
				BookingType:            tt.NewInt(1),
				PriorNoticeDurationMin: tt.NewInt(30),
				PriorNoticeLastDay:     tt.NewInt(1),
				PriorNoticeLastTime:    tt.NewSeconds(3600),
			},
			expectedErrors: ParseExpectErrors("ConditionallyForbiddenFieldError:prior_notice_last_day"),
		},

		// ===== BOOKING_TYPE=2 TESTS =====
		{
			name: "Invalid: booking_type=2 without prior_notice_last_day",
			bookingRule: &BookingRule{
				BookingRuleID: tt.NewString("rule12"),
				BookingType:   tt.NewInt(2),
			},
			expectedErrors: ParseExpectErrors("ConditionallyRequiredFieldError:prior_notice_last_day"),
		},
		{
			name: "Invalid: booking_type=2 with prior_notice_duration_min",
			bookingRule: &BookingRule{
				BookingRuleID:          tt.NewString("rule12_min"),
				BookingType:            tt.NewInt(2),
				PriorNoticeLastDay:     tt.NewInt(1),
				PriorNoticeLastTime:    tt.NewSeconds(3600),
				PriorNoticeDurationMin: tt.NewInt(30),
			},
			expectedErrors: ParseExpectErrors("ConditionallyForbiddenFieldError:prior_notice_duration_min"),
		},
		{
			name: "Valid: booking_type=2 with required fields",
			bookingRule: &BookingRule{
				BookingRuleID:       tt.NewString("rule13"),
				BookingType:         tt.NewInt(2),
				PriorNoticeLastDay:  tt.NewInt(1),
				PriorNoticeLastTime: tt.NewSeconds(3600),
			},
			expectedErrors: nil,
		},
		{
			name: "Invalid: booking_type=2 with prior_notice_duration_max",
			bookingRule: &BookingRule{
				BookingRuleID:          tt.NewString("rule14"),
				BookingType:            tt.NewInt(2),
				PriorNoticeLastDay:     tt.NewInt(1),
				PriorNoticeLastTime:    tt.NewSeconds(3600),
				PriorNoticeDurationMax: tt.NewInt(120),
			},
			expectedErrors: ParseExpectErrors("ConditionallyForbiddenFieldError:prior_notice_duration_max"),
		},
		{
			name: "Valid: booking_type=2 with prior_notice_service_id",
			bookingRule: &BookingRule{
				BookingRuleID:        tt.NewString("rule15"),
				BookingType:          tt.NewInt(2),
				PriorNoticeLastDay:   tt.NewInt(1),
				PriorNoticeLastTime:  tt.NewSeconds(3600),
				PriorNoticeServiceID: tt.NewKey("service1"),
			},
			expectedErrors: nil,
		},
		{
			name: "Invalid: booking_type=2 with both prior_notice_last_day missing and prior_notice_duration_max present",
			bookingRule: &BookingRule{
				BookingRuleID:          tt.NewString("rule16"),
				BookingType:            tt.NewInt(2),
				PriorNoticeDurationMax: tt.NewInt(120),
			},
			expectedErrors: ParseExpectErrors(
				"ConditionallyRequiredFieldError:prior_notice_last_day",
				"ConditionallyForbiddenFieldError:prior_notice_duration_max",
			),
		},

		// ===== TIME AND DAY DEPENDENCY TESTS =====
		{
			name: "Invalid: prior_notice_last_time without prior_notice_last_day",
			bookingRule: &BookingRule{
				BookingRuleID:          tt.NewString("rule17"),
				BookingType:            tt.NewInt(1),
				PriorNoticeDurationMin: tt.NewInt(30),
				PriorNoticeLastTime:    tt.NewSeconds(3600),
			},
			expectedErrors: ParseExpectErrors(
				"ConditionallyRequiredFieldError:prior_notice_last_day",
			),
		},
		{
			name: "Invalid: prior_notice_last_day without prior_notice_last_time",
			bookingRule: &BookingRule{
				BookingRuleID:      tt.NewString("rule17_missing_time"),
				BookingType:        tt.NewInt(2),
				PriorNoticeLastDay: tt.NewInt(1),
			},
			expectedErrors: ParseExpectErrors(
				"ConditionallyRequiredFieldError:prior_notice_last_time",
			),
		},
		{
			name: "Valid: prior_notice_last_time with prior_notice_last_day",
			bookingRule: &BookingRule{
				BookingRuleID:       tt.NewString("rule18"),
				BookingType:         tt.NewInt(2),
				PriorNoticeLastDay:  tt.NewInt(1),
				PriorNoticeLastTime: tt.NewSeconds(3600),
			},
			expectedErrors: nil,
		},
		{
			name: "Invalid: prior_notice_start_time without prior_notice_start_day",
			bookingRule: &BookingRule{
				BookingRuleID:          tt.NewString("rule19"),
				BookingType:            tt.NewInt(1),
				PriorNoticeDurationMin: tt.NewInt(30),
				PriorNoticeStartTime:   tt.NewSeconds(7200),
			},
			expectedErrors: ParseExpectErrors(
				"ConditionallyRequiredFieldError:prior_notice_start_day",
			),
		},
		{
			name: "Invalid: prior_notice_start_day without prior_notice_start_time",
			bookingRule: &BookingRule{
				BookingRuleID:          tt.NewString("rule19_missing_time"),
				BookingType:            tt.NewInt(1),
				PriorNoticeDurationMin: tt.NewInt(30),
				PriorNoticeStartDay:    tt.NewInt(1),
			},
			expectedErrors: ParseExpectErrors(
				"ConditionallyRequiredFieldError:prior_notice_start_time",
			),
		},
		{
			name: "Valid: prior_notice_start_time with prior_notice_start_day",
			bookingRule: &BookingRule{
				BookingRuleID:          tt.NewString("rule20"),
				BookingType:            tt.NewInt(1),
				PriorNoticeDurationMin: tt.NewInt(30),
				PriorNoticeStartDay:    tt.NewInt(2),
				PriorNoticeStartTime:   tt.NewSeconds(7200),
			},
			expectedErrors: nil,
		},

		// ===== COMPLEX SCENARIOS =====
		{
			name: "Valid: booking_type=2 with all compatible optional fields",
			bookingRule: &BookingRule{
				BookingRuleID:        tt.NewString("rule21"),
				BookingType:          tt.NewInt(2),
				PriorNoticeLastDay:   tt.NewInt(3),
				PriorNoticeLastTime:  tt.NewSeconds(3600),
				PriorNoticeStartDay:  tt.NewInt(1),
				PriorNoticeStartTime: tt.NewSeconds(7200),
				PriorNoticeServiceID: tt.NewKey("service1"),
				Message:              tt.NewString("Please book in advance"),
				PhoneNumber:          tt.NewString("555-1234"),
			},
			expectedErrors: nil,
		},
		{
			name: "Valid: booking_type=1 with all compatible optional fields",
			bookingRule: &BookingRule{
				BookingRuleID:          tt.NewString("rule22"),
				BookingType:            tt.NewInt(1),
				PriorNoticeDurationMin: tt.NewInt(15),
				PriorNoticeDurationMax: tt.NewInt(90),
				Message:                tt.NewString("Advanced booking required"),
				PickupMessage:          tt.NewString("Call when ready"),
				DropOffMessage:         tt.NewString("Confirm drop-off"),
			},
			expectedErrors: nil,
		},
		{
			name: "Valid: booking_type=0 with all compatible optional fields",
			bookingRule: &BookingRule{
				BookingRuleID: tt.NewString("rule23"),
				BookingType:   tt.NewInt(0),
				// PriorNoticeDurationMin removed as it is forbidden for type 0
				Message:    tt.NewString("Same-day service"),
				InfoURL:    tt.NewUrl("https://example.com/info"),
				BookingURL: tt.NewUrl("https://example.com/book"),
			},
			expectedErrors: nil,
		},
		{
			name: "Invalid: Multiple errors across different rules",
			bookingRule: &BookingRule{
				BookingRuleID:          tt.NewString("rule24"),
				BookingType:            tt.NewInt(1),
				PriorNoticeDurationMax: tt.NewInt(120),
				PriorNoticeStartDay:    tt.NewInt(1),
				PriorNoticeStartTime:   tt.NewSeconds(3600),
				PriorNoticeServiceID:   tt.NewKey("service1"),
			},
			expectedErrors: ParseExpectErrors(
				"ConditionallyRequiredFieldError:prior_notice_duration_min",
				"ConditionallyForbiddenFieldError:prior_notice_service_id",
				"ConditionallyForbiddenFieldError:prior_notice_start_day",
			),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tc.bookingRule.ConditionalErrors()
			CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}
