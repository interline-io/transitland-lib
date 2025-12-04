package rules

import (
	"fmt"
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

func TestFlexStopLocationTypeCheck(t *testing.T) {
	emap := tt.NewEntityMap()

	tests := []struct {
		name        string
		stop        *gtfs.Stop
		stopTime    *gtfs.StopTime
		expectError bool
	}{
		{
			name: "Valid: location_type=0 with flex service",
			stop: &gtfs.Stop{
				StopID:       tt.NewString("stop1"),
				LocationType: tt.NewInt(0),
			},
			stopTime: &gtfs.StopTime{
				StopID:     tt.NewKey("stop1"),
				PickupType: tt.NewInt(2), // flex service
			},
			expectError: false,
		},
		{
			name: "Invalid: location_type=1 (station) with flex service",
			stop: &gtfs.Stop{
				StopID:       tt.NewString("stop2"),
				LocationType: tt.NewInt(1),
			},
			stopTime: &gtfs.StopTime{
				StopID:     tt.NewKey("stop2"),
				PickupType: tt.NewInt(2),
			},
			expectError: true,
		},
		{
			name: "Valid: location_type=1 without flex service",
			stop: &gtfs.Stop{
				StopID:       tt.NewString("stop3"),
				LocationType: tt.NewInt(1),
			},
			stopTime: &gtfs.StopTime{
				StopID:     tt.NewKey("stop3"),
				PickupType: tt.NewInt(0), // regular service
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			check := &FlexStopLocationTypeCheck{}

			// Important: Process entities in order - stops first, then stop_times
			// This mimics how the copier processes entities

			// 1. Add the stop
			tc.stop.ID = 1
			check.AfterWrite(tc.stop.StopID.Val, tc.stop, emap)

			// 2. Add the stop_time (this marks the stop as flex if applicable)
			check.AfterWrite(tc.stopTime.StopID.Val, tc.stopTime, emap)

			// 3. Validate the stop (this checks if flex stops have correct location_type)
			errs := check.Validate(tc.stop)

			if tc.expectError {
				assert.NotEmpty(t, errs, "Expected validation error")
				if len(errs) > 0 {
					_, ok := errs[0].(*FlexStopLocationTypeError)
					assert.True(t, ok, "Expected FlexStopLocationTypeError")
				}
			} else {
				assert.Empty(t, errs, "Expected no validation errors")
			}
		})
	}
}

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

func TestFlexZoneIDConditionalCheck(t *testing.T) {
	emap := tt.NewEntityMap()

	tests := []struct {
		name        string
		location    *gtfs.Location
		fareRule    *gtfs.FareRule
		expectError bool
	}{
		{
			name: "Valid: zone_id present with fare_rules",
			location: &gtfs.Location{
				LocationID: tt.NewString("loc1"),
				ZoneID:     tt.NewString("zone1"),
			},
			fareRule:    &gtfs.FareRule{FareID: tt.NewString("fare1")},
			expectError: false,
		},
		{
			name: "Invalid: zone_id missing with fare_rules",
			location: &gtfs.Location{
				LocationID: tt.NewString("loc2"),
				ZoneID:     tt.String{},
			},
			fareRule:    &gtfs.FareRule{FareID: tt.NewString("fare1")},
			expectError: true,
		},
		{
			name: "Valid: zone_id missing without fare_rules",
			location: &gtfs.Location{
				LocationID: tt.NewString("loc3"),
				ZoneID:     tt.String{},
			},
			fareRule:    nil,
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			check := &FlexZoneIDConditionalCheck{}

			// Add fare_rule if present
			if tc.fareRule != nil {
				tc.fareRule.ID = 1
				check.AfterWrite("1", tc.fareRule, emap)
			}

			// Add the location
			tc.location.ID = 2
			check.AfterWrite("2", tc.location, emap)

			// Validate
			errs := check.Validate(tc.location)

			if tc.expectError {
				assert.NotEmpty(t, errs, "Expected validation error")
				if len(errs) > 0 {
					_, ok := errs[0].(*FlexZoneIDRequiredError)
					assert.True(t, ok, "Expected FlexZoneIDRequiredError")
				}
			} else {
				assert.Empty(t, errs, "Expected no validation errors")
			}
		})
	}
}

func TestFlexGeographyIDUniqueCheckIntegration(t *testing.T) {
	emap := tt.NewEntityMap()

	t.Run("duplicate stop_id with location_group_id", func(t *testing.T) {
		check := &FlexGeographyIDUniqueCheck{}

		// Create a stop
		stop := &gtfs.Stop{
			StopID:   tt.NewString("stop1"),
			StopName: tt.NewString("Test Stop"),
		}
		stop.ID = 1

		// Create a location_group with the same ID
		locationGroup := &gtfs.LocationGroup{
			LocationGroupID:   tt.NewString("stop1"),
			LocationGroupName: tt.NewString("Duplicate ID"),
		}
		locationGroup.ID = 2

		// Process entities
		check.AfterWrite("1", stop, emap)
		check.AfterWrite("2", locationGroup, emap)

		// Validate - should detect duplicate
		errs := check.Validate(locationGroup)
		assert.NotEmpty(t, errs, "Expected duplicate geography ID error")
		if len(errs) > 0 {
			_, ok := errs[0].(*FlexGeographyIDDuplicateError)
			assert.True(t, ok, "Expected FlexGeographyIDDuplicateError")
			assert.Contains(t, errs[0].Error(), "stop1")
			assert.Contains(t, errs[0].Error(), "stops.txt")
			assert.Contains(t, errs[0].Error(), "location_groups.txt")
		}
	})

	t.Run("duplicate stop_id with location_id", func(t *testing.T) {
		check := &FlexGeographyIDUniqueCheck{}

		// Create a stop
		stop := &gtfs.Stop{
			StopID:   tt.NewString("area_1"),
			StopName: tt.NewString("Test Stop"),
		}
		stop.ID = 1

		// Create a location with the same ID
		location := &gtfs.Location{
			LocationID: tt.NewString("area_1"),
			StopName:   tt.NewString("Duplicate ID"),
		}
		location.ID = 2

		// Process entities
		check.AfterWrite("1", stop, emap)
		check.AfterWrite("2", location, emap)

		// Validate - should detect duplicate
		errs := check.Validate(location)
		assert.NotEmpty(t, errs, "Expected duplicate geography ID error")
		if len(errs) > 0 {
			_, ok := errs[0].(*FlexGeographyIDDuplicateError)
			assert.True(t, ok, "Expected FlexGeographyIDDuplicateError")
			assert.Contains(t, errs[0].Error(), "area_1")
			assert.Contains(t, errs[0].Error(), "stops.txt")
			assert.Contains(t, errs[0].Error(), "locations.geojson")
		}
	})

	t.Run("duplicate location_group_id with location_id", func(t *testing.T) {
		check := &FlexGeographyIDUniqueCheck{}

		// Create a location_group
		locationGroup := &gtfs.LocationGroup{
			LocationGroupID:   tt.NewString("zone_1"),
			LocationGroupName: tt.NewString("Zone 1"),
		}
		locationGroup.ID = 1

		// Create a location with the same ID
		location := &gtfs.Location{
			LocationID: tt.NewString("zone_1"),
			StopName:   tt.NewString("Duplicate ID"),
		}
		location.ID = 2

		// Process entities
		check.AfterWrite("1", locationGroup, emap)
		check.AfterWrite("2", location, emap)

		// Validate - should detect duplicate
		errs := check.Validate(location)
		assert.NotEmpty(t, errs, "Expected duplicate geography ID error")
		if len(errs) > 0 {
			_, ok := errs[0].(*FlexGeographyIDDuplicateError)
			assert.True(t, ok, "Expected FlexGeographyIDDuplicateError")
			assert.Contains(t, errs[0].Error(), "zone_1")
			assert.Contains(t, errs[0].Error(), "location_groups.txt")
			assert.Contains(t, errs[0].Error(), "locations.geojson")
		}
	})
}
