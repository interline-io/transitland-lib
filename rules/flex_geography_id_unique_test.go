package rules

import (
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

func TestFlexGeographyIDUniqueCheck(t *testing.T) {
	tests := []struct {
		name        string
		entities    []tt.Entity
		expectError bool
		errorMsg    string
	}{
		{
			name: "no duplicates - all unique",
			entities: []tt.Entity{
				&gtfs.Stop{StopID: tt.NewString("stop_1")},
				&gtfs.Stop{StopID: tt.NewString("stop_2")},
				&gtfs.Location{LocationID: tt.NewString("location_1")},
				&gtfs.LocationGroup{LocationGroupID: tt.NewString("group_1")},
			},
			expectError: false,
		},
		{
			name: "duplicate stop_id with location_id",
			entities: []tt.Entity{
				&gtfs.Stop{StopID: tt.NewString("area_1")},
				&gtfs.Location{LocationID: tt.NewString("area_1")},
			},
			expectError: true,
			errorMsg:    "geography_id 'area_1' is duplicated across stops.txt and locations.geojson (IDs must be unique across stops.stop_id, locations.geojson id, and location_groups.location_group_id)",
		},
		{
			name: "duplicate stop_id with location_group_id",
			entities: []tt.Entity{
				&gtfs.Stop{StopID: tt.NewString("zone_5")},
				&gtfs.LocationGroup{LocationGroupID: tt.NewString("zone_5")},
			},
			expectError: true,
			errorMsg:    "geography_id 'zone_5' is duplicated across stops.txt and location_groups.txt (IDs must be unique across stops.stop_id, locations.geojson id, and location_groups.location_group_id)",
		},
		{
			name: "duplicate location_id with location_group_id",
			entities: []tt.Entity{
				&gtfs.Location{LocationID: tt.NewString("flex_area")},
				&gtfs.LocationGroup{LocationGroupID: tt.NewString("flex_area")},
			},
			expectError: true,
			errorMsg:    "geography_id 'flex_area' is duplicated across locations.geojson and location_groups.txt (IDs must be unique across stops.stop_id, locations.geojson id, and location_groups.location_group_id)",
		},
		{
			name: "multiple stops with same ID within stops.txt (should pass this check)",
			entities: []tt.Entity{
				&gtfs.Stop{StopID: tt.NewString("stop_1")},
				&gtfs.Stop{StopID: tt.NewString("stop_1")},
			},
			expectError: false, // This check doesn't validate duplicates within the same file
		},
		{
			name: "empty IDs (should not trigger error)",
			entities: []tt.Entity{
				&gtfs.Stop{StopID: tt.NewString("")},
				&gtfs.Location{LocationID: tt.NewString("")},
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			check := &FlexGeographyIDUniqueCheck{}
			emap := tt.NewEntityMap()

			// First pass: AfterWrite to collect IDs
			for _, ent := range tc.entities {
				err := check.AfterWrite("", ent, emap)
				if err != nil {
					t.Fatalf("AfterWrite failed: %v", err)
				}
			}

			// Second pass: Validate to detect duplicates
			var foundError bool
			var errorMessage string
			for _, ent := range tc.entities {
				errs := check.Validate(ent)
				if len(errs) > 0 {
					foundError = true
					errorMessage = errs[0].Error()
					break
				}
			}

			if tc.expectError && !foundError {
				t.Errorf("Expected error but got none")
			}
			if !tc.expectError && foundError {
				t.Errorf("Expected no error but got: %s", errorMessage)
			}
			if tc.expectError && foundError && tc.errorMsg != "" {
				if errorMessage != tc.errorMsg {
					t.Errorf("Expected error message:\n  %q\nGot:\n  %q", tc.errorMsg, errorMessage)
				}
			}
		})
	}
}

