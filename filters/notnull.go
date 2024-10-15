package filters

import (
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// NotNullFilter sets some int fields to default zero values
// where there is a 'not null' database constraint that can't be removed
// but the spec allows empty values.
type NotNullFilter struct{}

// Validate .
func (e *NotNullFilter) Filter(ent tt.Entity, emap *tt.EntityMap) error {
	switch v := ent.(type) {
	case *gtfs.Stop:
		if !v.LocationType.Valid {
			v.LocationType.Set(0)
		}
	case *gtfs.Trip:
		if !v.DirectionID.Valid {
			v.DirectionID.Set(0)
		}
		if !v.StopPatternID.Valid {
			v.StopPatternID.Set(0)
		}
		if !v.JourneyPatternOffset.Valid {
			v.JourneyPatternOffset.Set(0)
		}
	}
	return nil
}
