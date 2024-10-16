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
		// Used in textsearch
		v.AgencyID.OrSet("")
		v.AgencyURL.OrSet("")
		v.AgencyEmail.OrSet("")
		v.AgencyURL.OrSet("")
	case *gtfs.Route:
		// Used in textsearch
		v.RouteLongName.OrSet("")
		v.RouteShortName.OrSet("")
		v.RouteDesc.OrSet("")
		v.RouteID.OrSet("")
	case *gtfs.Stop:
		v.LocationType.OrSet(0)
		// Used in textsearch
		v.StopName.OrSet("")
		v.StopDesc.OrSet("")
		v.StopCode.OrSet("")
		v.StopURL.OrSet("")
		v.StopID.OrSet("")
	case *gtfs.Trip:
		v.DirectionID.OrSet(0)
		v.StopPatternID.OrSet(0)
		v.JourneyPatternOffset.OrSet(0)
	case *gtfs.Transfer:
		v.TransferType.OrSet(0)
	}
	return nil
}
