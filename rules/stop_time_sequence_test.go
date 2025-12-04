package rules

import (
	"strconv"
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tt"
)

// Test helpers for building stop_time sequences with various configurations.
// These helpers make it easy to construct test cases for both scheduled GTFS
// and GTFS-Flex trips with time windows.

// Helper to create a basic stop_time with common defaults
func makeStopTime(stopSeq int, arrivalTime, departureTime int) gtfs.StopTime {
	return gtfs.StopTime{
		TripID:        tt.NewString("trip1"),
		StopID:        tt.NewKey(strconv.Itoa(stopSeq)),
		StopSequence:  tt.NewInt(stopSeq),
		ArrivalTime:   tt.NewSeconds(arrivalTime),
		DepartureTime: tt.NewSeconds(departureTime),
	}
}

// Helper to create a flex stop_time with time windows
func makeFlexStopTime(stopSeq int, startWindow, endWindow int) gtfs.StopTime {
	st := gtfs.StopTime{
		TripID:       tt.NewString("trip1"),
		StopID:       tt.NewKey(strconv.Itoa(stopSeq)),
		StopSequence: tt.NewInt(stopSeq),
	}
	if startWindow > 0 {
		st.StartPickupDropOffWindow = tt.NewSeconds(startWindow)
	}
	if endWindow > 0 {
		st.EndPickupDropOffWindow = tt.NewSeconds(endWindow)
	}
	return st
}

// Helper to add shape distance to a stop_time
func withShapeDist(st gtfs.StopTime, dist float64) gtfs.StopTime {
	st.ShapeDistTraveled = tt.NewFloat(dist)
	return st
}

// Helper to override stop sequence
func withStopSequence(st gtfs.StopTime, seq int) gtfs.StopTime {
	st.StopSequence = tt.NewInt(seq)
	return st
}

func TestValidateStopTimes(t *testing.T) {
	tests := []struct {
		name           string
		stopTimes      []gtfs.StopTime
		expectedErrors []testutil.ExpectError
	}{
		// ===== VALID SCHEDULED TRIPS =====
		{
			name: "valid_basic_scheduled_trip",
			stopTimes: []gtfs.StopTime{
				makeStopTime(0, 1000, 1000),
				makeStopTime(1, 2000, 2000),
				makeStopTime(2, 3000, 3000),
			},
			expectedErrors: nil,
		},
		{
			name: "valid_missing_intermediate_times",
			stopTimes: []gtfs.StopTime{
				makeStopTime(0, 1000, 1000),
				makeStopTime(1, 0, 0), // intermediate can be missing
				makeStopTime(2, 3000, 3000),
			},
			expectedErrors: nil,
		},
		{
			name: "valid_first_missing_arrival",
			stopTimes: []gtfs.StopTime{
				makeStopTime(0, 0, 1000), // first arrival_time can be missing
				makeStopTime(1, 2000, 2000),
			},
			expectedErrors: nil,
		},
		{
			name: "valid_last_missing_departure",
			stopTimes: []gtfs.StopTime{
				makeStopTime(0, 1000, 1000),
				makeStopTime(1, 2000, 0), // last departure_time can be missing
			},
			expectedErrors: nil,
		},
		{
			name: "valid_with_shape_distances",
			stopTimes: []gtfs.StopTime{
				withShapeDist(makeStopTime(0, 1000, 1000), 0.0),
				withShapeDist(makeStopTime(1, 2000, 2000), 5.5),
				withShapeDist(makeStopTime(2, 3000, 3000), 12.3),
			},
			expectedErrors: nil,
		},
		{
			name: "valid_no_shape_distances",
			stopTimes: []gtfs.StopTime{
				makeStopTime(0, 1000, 1000),
				makeStopTime(1, 2000, 2000),
				makeStopTime(2, 3000, 3000),
			},
			expectedErrors: nil,
		},
		{
			name: "valid_partial_shape_distances",
			stopTimes: []gtfs.StopTime{
				makeStopTime(0, 1000, 1000),
				withShapeDist(makeStopTime(1, 2000, 2000), 5.5),
				withShapeDist(makeStopTime(2, 3000, 3000), 12.3),
			},
			expectedErrors: nil,
		},

		// ===== VALID FLEX TRIPS =====
		{
			name: "valid_flex_trip_start_window",
			stopTimes: []gtfs.StopTime{
				makeFlexStopTime(0, 1000, 0), // start_window only
				makeFlexStopTime(1, 2000, 0),
				makeFlexStopTime(2, 3000, 0),
			},
			expectedErrors: nil,
		},
		{
			name: "valid_flex_trip_end_window",
			stopTimes: []gtfs.StopTime{
				makeFlexStopTime(0, 0, 1000), // end_window only
				makeFlexStopTime(1, 0, 2000),
				makeFlexStopTime(2, 0, 3000),
			},
			expectedErrors: nil,
		},
		{
			name: "valid_flex_trip_both_windows",
			stopTimes: []gtfs.StopTime{
				makeFlexStopTime(0, 1000, 2000), // both windows
				makeFlexStopTime(1, 2000, 3000),
				makeFlexStopTime(2, 3000, 4000),
			},
			expectedErrors: nil,
		},

		// ===== VALID MIXED TRIPS =====
		{
			name: "valid_mixed_scheduled_then_flex",
			stopTimes: []gtfs.StopTime{
				makeStopTime(0, 1000, 1000),     // scheduled
				makeStopTime(1, 2000, 2000),     // scheduled
				makeFlexStopTime(2, 3000, 4000), // flex
			},
			expectedErrors: nil,
		},
		{
			name: "valid_mixed_flex_then_scheduled",
			stopTimes: []gtfs.StopTime{
				makeFlexStopTime(0, 1000, 2000), // flex
				makeStopTime(1, 3000, 3000),     // scheduled
				makeStopTime(2, 4000, 4000),     // scheduled
			},
			expectedErrors: nil,
		},
		{
			name: "valid_mixed_interleaved",
			stopTimes: []gtfs.StopTime{
				makeStopTime(0, 1000, 1000),     // scheduled
				makeFlexStopTime(1, 2000, 3000), // flex
				makeStopTime(2, 4000, 4000),     // scheduled
			},
			expectedErrors: nil,
		},

		// ===== ERRORS: EMPTY/TOO FEW STOPS =====
		{
			name:           "error_empty_trip",
			stopTimes:      []gtfs.StopTime{},
			expectedErrors: testutil.ParseExpectErrors("EmptyTripError"),
		},
		{
			name: "error_single_stop",
			stopTimes: []gtfs.StopTime{
				makeStopTime(0, 1000, 1000),
			},
			expectedErrors: testutil.ParseExpectErrors("EmptyTripError"),
		},

		// ===== ERRORS: LAST STOP =====
		{
			name: "error_last_stop_no_arrival_or_window",
			stopTimes: []gtfs.StopTime{
				makeStopTime(0, 1000, 1000),
				makeStopTime(1, 0, 2000), // missing arrival_time and no time window
			},
			expectedErrors: testutil.ParseExpectErrors("SequenceError:arrival_time"),
		},

		// ===== ERRORS: STOP SEQUENCE =====
		{
			name: "error_duplicate_stop_sequence",
			stopTimes: []gtfs.StopTime{
				makeStopTime(1, 1000, 1000),
				makeStopTime(2, 2000, 2000),
				withStopSequence(makeStopTime(2, 3000, 3000), 2), // duplicate sequence
			},
			expectedErrors: testutil.ParseExpectErrors("SequenceError:stop_sequence"),
		},
		{
			name: "error_multiple_duplicate_sequences",
			stopTimes: []gtfs.StopTime{
				makeStopTime(5, 1000, 1000),
				withStopSequence(makeStopTime(1, 2000, 2000), 5), // duplicate
				makeStopTime(10, 3000, 3000),
				withStopSequence(makeStopTime(3, 4000, 4000), 10), // duplicate
			},
			expectedErrors: testutil.ParseExpectErrors(
				"SequenceError:stop_sequence",
				"SequenceError:stop_sequence",
			),
		},

		// ===== ERRORS: TIME PROGRESSION (SCHEDULED STOPS ONLY) =====
		{
			name: "error_arrival_before_previous_departure",
			stopTimes: []gtfs.StopTime{
				makeStopTime(0, 1000, 1000),
				makeStopTime(1, 500, 2000), // arrival < previous departure
			},
			expectedErrors: testutil.ParseExpectErrors("SequenceError:arrival_time"),
		},
		{
			name: "error_departure_before_arrival_same_stop",
			stopTimes: []gtfs.StopTime{
				makeStopTime(0, 1000, 1000),
				makeStopTime(1, 2000, 1500), // departure < arrival at same stop
			},
			expectedErrors: testutil.ParseExpectErrors("SequenceError:departure_time"),
		},
		{
			name: "error_multiple_time_violations",
			stopTimes: []gtfs.StopTime{
				makeStopTime(0, 1000, 1000),
				makeStopTime(1, 500, 2000),  // arrival < previous departure
				makeStopTime(2, 3000, 2500), // departure < arrival
			},
			expectedErrors: testutil.ParseExpectErrors(
				"SequenceError:arrival_time",
				"SequenceError:departure_time",
			),
		},
		{
			name: "error_time_goes_backward",
			stopTimes: []gtfs.StopTime{
				makeStopTime(0, 1000, 1000),
				makeStopTime(1, 2000, 2000),
				makeStopTime(2, 1500, 1500), // times go backward
			},
			expectedErrors: testutil.ParseExpectErrors("SequenceError:arrival_time"),
		},

		// ===== NO ERRORS: TIME PROGRESSION WITH FLEX STOPS =====
		{
			name: "no_error_flex_stop_skips_time_validation",
			stopTimes: []gtfs.StopTime{
				makeStopTime(0, 1000, 1000),   // scheduled at 1000
				makeFlexStopTime(1, 100, 200), // flex with "earlier" window - should be ignored
				makeStopTime(2, 3000, 3000),   // scheduled at 3000 (compared to stop 0)
			},
			expectedErrors: nil,
		},
		{
			name: "no_error_mixed_trip_time_validation_skips_flex",
			stopTimes: []gtfs.StopTime{
				makeStopTime(0, 1000, 1000),   // scheduled
				makeFlexStopTime(1, 500, 600), // flex in between
				makeStopTime(2, 2000, 2000),   // scheduled (compared to stop 0, not stop 1)
			},
			expectedErrors: nil,
		},

		// ===== ERRORS: SHAPE DISTANCE =====
		{
			name: "error_shape_distance_decreases",
			stopTimes: []gtfs.StopTime{
				withShapeDist(makeStopTime(0, 1000, 1000), 10.0),
				withShapeDist(makeStopTime(1, 2000, 2000), 5.0), // distance decreases
			},
			expectedErrors: testutil.ParseExpectErrors("SequenceError:shape_dist_traveled"),
		},
		{
			name: "error_multiple_shape_distance_violations",
			stopTimes: []gtfs.StopTime{
				withShapeDist(makeStopTime(0, 1000, 1000), 10.0),
				withShapeDist(makeStopTime(1, 2000, 2000), 5.0), // decreases
				withShapeDist(makeStopTime(2, 3000, 3000), 15.0),
				withShapeDist(makeStopTime(3, 4000, 4000), 12.0), // decreases
			},
			expectedErrors: testutil.ParseExpectErrors(
				"SequenceError:shape_dist_traveled",
				"SequenceError:shape_dist_traveled",
			),
		},
		{
			name: "no_error_shape_distance_stays_same",
			stopTimes: []gtfs.StopTime{
				withShapeDist(makeStopTime(0, 1000, 1000), 10.0),
				withShapeDist(makeStopTime(1, 2000, 2000), 10.0), // same is OK
				withShapeDist(makeStopTime(2, 3000, 3000), 15.0),
			},
			expectedErrors: nil,
		},

		// ===== MULTIPLE ERROR TYPES =====
		{
			name: "error_combined_sequence_and_time",
			stopTimes: []gtfs.StopTime{
				makeStopTime(1, 1000, 1000),
				makeStopTime(1, 500, 2000),                       // duplicate sequence + time violation
				withStopSequence(makeStopTime(2, 3000, 3000), 1), // another duplicate sequence
			},
			expectedErrors: testutil.ParseExpectErrors(
				"SequenceError:stop_sequence",
				"SequenceError:arrival_time",
				"SequenceError:stop_sequence",
			),
		},
		{
			name: "error_all_validation_types",
			stopTimes: []gtfs.StopTime{
				withShapeDist(makeStopTime(1, 1000, 1000), 10.0),                  // first stop is valid now
				withShapeDist(makeStopTime(1, 2000, 2000), 5.0),                   // duplicate sequence + shape decreases
				withShapeDist(withStopSequence(makeStopTime(2, 0, 3000), 1), 3.0), // another duplicate sequence + missing arrival (last stop) + shape decreases
			},
			expectedErrors: testutil.ParseExpectErrors(
				"SequenceError:arrival_time",
				"SequenceError:stop_sequence",
				"SequenceError:shape_dist_traveled",
				"SequenceError:stop_sequence",
				"SequenceError:shape_dist_traveled",
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateStopTimes(tt.stopTimes)
			testutil.CheckErrors(tt.expectedErrors, errs, t)
		})
	}
}

func BenchmarkValidateStopTime(b *testing.B) {
	stoptimes := []gtfs.StopTime{
		makeStopTime(0, 10, 10),
		makeStopTime(1, 20, 20),
		makeStopTime(2, 30, 30),
		makeStopTime(3, 40, 40),
		makeStopTime(4, 50, 50),
		makeStopTime(5, 60, 60),
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ValidateStopTimes(stoptimes)
	}
}
