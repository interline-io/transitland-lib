package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

func TestLocationGroup_Errors(t *testing.T) {
	tests := []struct {
		name           string
		locationGroup  *LocationGroup
		expectedErrors []ExpectError
	}{
		{
			name: "Valid: location_group with ID and name",
			locationGroup: &LocationGroup{
				LocationGroupID:   tt.NewString("lg1"),
				LocationGroupName: tt.NewString("Transit Mall"),
			},
			expectedErrors: nil,
		},
		{
			name: "Valid: location_group with ID only",
			locationGroup: &LocationGroup{
				LocationGroupID: tt.NewString("lg2"),
			},
			expectedErrors: nil,
		},
		{
			name: "Invalid: location_group without required location_group_id",
			locationGroup: &LocationGroup{
				LocationGroupName: tt.NewString("Transit Mall"),
			},
			expectedErrors: ParseExpectErrors("RequiredFieldError:location_group_id"),
		},
		{
			name: "Valid: location_group with special characters in ID",
			locationGroup: &LocationGroup{
				LocationGroupID:   tt.NewString("lg_zone-1:downtown"),
				LocationGroupName: tt.NewString("Zone 1: Downtown"),
			},
			expectedErrors: nil,
		},
		{
			name: "Valid: location_group with unicode name",
			locationGroup: &LocationGroup{
				LocationGroupID:   tt.NewString("lg_unicode"),
				LocationGroupName: tt.NewString("市中心區域 (Downtown)"),
			},
			expectedErrors: nil,
		},
		{
			name: "Valid: location_group with long name",
			locationGroup: &LocationGroup{
				LocationGroupID:   tt.NewString("lg_long"),
				LocationGroupName: tt.NewString("This is a very long location group name that describes a comprehensive service area covering multiple neighborhoods and districts"),
			},
			expectedErrors: nil,
		},
		{
			name: "Valid: location_group with empty name (optional field)",
			locationGroup: &LocationGroup{
				LocationGroupID: tt.NewString("lg_no_name"),
			},
			expectedErrors: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.locationGroup)
			CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}

func TestLocationGroup_EntityMethods(t *testing.T) {
	lg := LocationGroup{
		LocationGroupID:   tt.NewString("test_lg"),
		LocationGroupName: tt.NewString("Test Group"),
	}

	if key := lg.EntityKey(); key != "test_lg" {
		t.Errorf("EntityKey() = %q, want %q", key, "test_lg")
	}

	if filename := lg.Filename(); filename != "location_groups.txt" {
		t.Errorf("Filename() = %q, want %q", filename, "location_groups.txt")
	}

	if table := lg.TableName(); table != "gtfs_location_groups" {
		t.Errorf("TableName() = %q, want %q", table, "gtfs_location_groups")
	}

	// Test EntityID with no ID set
	if entityID := lg.EntityID(); entityID != "test_lg" {
		t.Errorf("EntityID() without ID = %q, want %q", entityID, "test_lg")
	}

	// Test EntityID with ID set
	lg.ID = 123
	if entityID := lg.EntityID(); entityID != "123" {
		t.Errorf("EntityID() with ID = %q, want %q", entityID, "123")
	}
}
