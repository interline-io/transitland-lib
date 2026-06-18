package bestpractices

import (
	"fmt"
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

func TestFlexLocationGroupEmptyCheck(t *testing.T) {
	emap := tt.NewEntityMap()

	tests := []struct {
		name        string
		group       *gtfs.LocationGroup
		stops       []*gtfs.LocationGroupStop
		expectError bool
	}{
		{
			name: "Valid: location_group with stops",
			group: &gtfs.LocationGroup{
				LocationGroupID: tt.NewString("group1"),
			},
			stops: []*gtfs.LocationGroupStop{
				{LocationGroupID: tt.NewKey("group1"), StopID: tt.NewKey("stop1")},
				{LocationGroupID: tt.NewKey("group1"), StopID: tt.NewKey("stop2")},
			},
			expectError: false,
		},
		{
			name: "Invalid: location_group with no stops",
			group: &gtfs.LocationGroup{
				LocationGroupID: tt.NewString("group2"),
			},
			stops:       []*gtfs.LocationGroupStop{},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			check := &FlexLocationGroupEmptyCheck{}

			// Add the location_group
			tc.group.ID = 1
			check.AfterWrite("1", tc.group, emap)

			// Add the location_group_stops
			for i, lgs := range tc.stops {
				lgs.ID = i + 2
				check.AfterWrite(fmt.Sprintf("%d", i+2), lgs, emap)
			}

			// Validate
			errs := check.Validate(tc.group)

			if tc.expectError {
				assert.NotEmpty(t, errs, "Expected validation error")
				if len(errs) > 0 {
					_, ok := errs[0].(*FlexLocationGroupEmptyError)
					assert.True(t, ok, "Expected FlexLocationGroupEmptyError")
				}
			} else {
				assert.Empty(t, errs, "Expected no validation errors")
			}
		})
	}
}
