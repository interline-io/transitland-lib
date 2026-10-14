package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// FlexZoneIDRequiredError reports when zone_id is missing in locations.geojson but fare_rules.txt exists.
type FlexZoneIDRequiredError struct {
	LocationID string
	bc
}

func (e *FlexZoneIDRequiredError) Error() string {
	return fmt.Sprintf(
		"location '%s' requires zone_id when fare_rules.txt is defined",
		e.LocationID,
	)
}

// FlexZoneIDConditionalCheck validates that locations have zone_id when fare_rules.txt exists.
type FlexZoneIDConditionalCheck struct {
	hasFareRules bool
	checkedFiles bool
}

func (e *FlexZoneIDConditionalCheck) AfterWrite(eid string, ent tt.Entity, _ *tt.EntityMap) error {
	// Check if fare_rules.txt has any entries
	if _, ok := ent.(*gtfs.FareRule); ok {
		e.hasFareRules = true
	}
	e.checkedFiles = true
	return nil
}

func (e *FlexZoneIDConditionalCheck) Validate(ent tt.Entity) []error {
	if !e.checkedFiles {
		return nil
	}

	loc, ok := ent.(*gtfs.Location)
	if !ok {
		return nil
	}

	// If fare_rules.txt exists, zone_id is required in locations
	if e.hasFareRules && !loc.ZoneID.Valid {
		return []error{&FlexZoneIDRequiredError{
			LocationID: loc.LocationID.Val,
		}}
	}

	return nil
}
