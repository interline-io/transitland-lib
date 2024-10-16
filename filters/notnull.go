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
	case *gtfs.Agency:
		v.AgencyID.OrSet("")
	case *gtfs.Stop:
		v.LocationType.OrSet(0)
	case *gtfs.Trip:
		v.DirectionID.OrSet(0)
		v.StopPatternID.OrSet(0)
		v.JourneyPatternOffset.OrSet(0)
	case *gtfs.Transfer:
		v.TransferType.OrSet(0)
	}
	return nil
}
