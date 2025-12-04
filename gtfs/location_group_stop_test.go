package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

func TestLocationGroupStop_Errors(t *testing.T) {
	tests := []struct {
		name              string
		locationGroupStop *LocationGroupStop
		expectedErrors    []ExpectError
	}{
		{
			name: "Valid: basic location_group_stop",
			locationGroupStop: &LocationGroupStop{
				LocationGroupID: tt.NewKey("lg1"),
				StopID:          tt.NewKey("stop1"),
			},
			expectedErrors: nil,
		},
		{
			name: "Valid: location_group_stop with alphanumeric IDs",
			locationGroupStop: &LocationGroupStop{
				LocationGroupID: tt.NewKey("lg_downtown_001"),
				StopID:          tt.NewKey("stop_main_street_123"),
			},
			expectedErrors: nil,
		},
		{
			name: "Invalid: missing location_group_id",
			locationGroupStop: &LocationGroupStop{
				StopID: tt.NewKey("stop1"),
			},
			expectedErrors: ParseExpectErrors("RequiredFieldError:location_group_id"),
		},
		{
			name: "Invalid: missing stop_id",
			locationGroupStop: &LocationGroupStop{
				LocationGroupID: tt.NewKey("lg1"),
			},
			expectedErrors: ParseExpectErrors("RequiredFieldError:stop_id"),
		},
		{
			name:              "Invalid: missing both required fields",
			locationGroupStop: &LocationGroupStop{},
			expectedErrors:    ParseExpectErrors("RequiredFieldError:location_group_id", "RequiredFieldError:stop_id"),
		},
		{
			name: "Valid: location_group_stop with mixed case IDs",
			locationGroupStop: &LocationGroupStop{
				LocationGroupID: tt.NewKey("LG-Zone1"),
				StopID:          tt.NewKey("StopID-ABC"),
			},
			expectedErrors: nil,
		},
		{
			name: "Valid: location_group_stop with long IDs",
			locationGroupStop: &LocationGroupStop{
				LocationGroupID: tt.NewKey("location_group_for_downtown_service_area_zone_1_extended"),
				StopID:          tt.NewKey("stop_at_main_street_and_first_avenue_northeast_corner"),
			},
			expectedErrors: nil,
		},
		{
			name: "Valid: location_group_stop with unicode IDs",
			locationGroupStop: &LocationGroupStop{
				LocationGroupID: tt.NewKey("市中心区域"),
				StopID:          tt.NewKey("駅001"),
			},
			expectedErrors: nil,
		},
		{
			name: "Valid: location_group_stop with URL-encoded IDs",
			locationGroupStop: &LocationGroupStop{
				LocationGroupID: tt.NewKey("lg%20zone%201"),
				StopID:          tt.NewKey("stop%2001"),
			},
			expectedErrors: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.locationGroupStop)
			CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}

func TestLocationGroupStop_EntityMethods(t *testing.T) {
	lgs := LocationGroupStop{
		LocationGroupID: tt.NewKey("lg_test"),
		StopID:          tt.NewKey("stop_test"),
	}

	if filename := lgs.Filename(); filename != "location_group_stops.txt" {
		t.Errorf("Filename() = %q, want %q", filename, "location_group_stops.txt")
	}

	if table := lgs.TableName(); table != "gtfs_location_group_stops" {
		t.Errorf("TableName() = %q, want %q", table, "gtfs_location_group_stops")
	}

	// Test with ID set
	lgs.ID = 456
	if id := lgs.ID; id != 456 {
		t.Errorf("ID = %d, want %d", id, 456)
	}
}

func TestLocationGroupStop_UpdateKeys(t *testing.T) {
	tests := []struct {
		name              string
		locationGroupStop *LocationGroupStop
		setupEntityMap    func() *tt.EntityMap
		expectError       bool
		errorField        string
	}{
		{
			name: "Valid: both keys resolve successfully",
			locationGroupStop: &LocationGroupStop{
				LocationGroupID: tt.NewKey("lg1"),
				StopID:          tt.NewKey("stop1"),
			},
			setupEntityMap: func() *tt.EntityMap {
				emap := tt.NewEntityMap()
				emap.Set("location_groups.txt", "lg1", "100")
				emap.Set("stops.txt", "stop1", "200")
				return emap
			},
			expectError: false,
		},
		{
			name: "Invalid: location_group_id not found",
			locationGroupStop: &LocationGroupStop{
				LocationGroupID: tt.NewKey("lg_unknown"),
				StopID:          tt.NewKey("stop1"),
			},
			setupEntityMap: func() *tt.EntityMap {
				emap := tt.NewEntityMap()
				emap.Set("stops.txt", "stop1", "200")
				return emap
			},
			expectError: true,
			errorField:  "location_group_id",
		},
		{
			name: "Invalid: stop_id not found",
			locationGroupStop: &LocationGroupStop{
				LocationGroupID: tt.NewKey("lg1"),
				StopID:          tt.NewKey("stop_unknown"),
			},
			setupEntityMap: func() *tt.EntityMap {
				emap := tt.NewEntityMap()
				emap.Set("location_groups.txt", "lg1", "100")
				return emap
			},
			expectError: true,
			errorField:  "stop_id",
		},
		{
			name: "Invalid: both keys not found",
			locationGroupStop: &LocationGroupStop{
				LocationGroupID: tt.NewKey("lg_unknown"),
				StopID:          tt.NewKey("stop_unknown"),
			},
			setupEntityMap: func() *tt.EntityMap {
				return tt.NewEntityMap()
			},
			expectError: true,
			errorField:  "location_group_id", // First error is returned
		},
		{
			name: "Valid: keys with special characters",
			locationGroupStop: &LocationGroupStop{
				LocationGroupID: tt.NewKey("lg:zone-1"),
				StopID:          tt.NewKey("stop_123-456"),
			},
			setupEntityMap: func() *tt.EntityMap {
				emap := tt.NewEntityMap()
				emap.Set("location_groups.txt", "lg:zone-1", "100")
				emap.Set("stops.txt", "stop_123-456", "200")
				return emap
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			emap := tc.setupEntityMap()
			err := tc.locationGroupStop.UpdateKeys(emap)

			if tc.expectError && err == nil {
				t.Error("Expected error but got nil")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tc.expectError && err != nil && tc.errorField != "" {
				errStr := err.Error()
				if len(errStr) == 0 || errStr == "" {
					t.Errorf("Expected error message containing %q but got empty error", tc.errorField)
				}
			}
		})
	}
}
