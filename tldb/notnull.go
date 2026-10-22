package tldb

import (
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// NotNullFilter sets the int and bool fields the spec allows to be empty, but
// whose database columns are not null, to their zero values. A zero reads back
// from the database as written, so readers see these defaults.
type NotNullFilter struct{}

// Filter sets an entity's not-null defaults.
func (e *NotNullFilter) Filter(ent tt.Entity, emap *tt.EntityMap) error {
	switch v := ent.(type) {
	case *gtfs.Stop:
		v.LocationType.OrSet(0)
	case *gtfs.Trip:
		v.DirectionID.OrSet(0)
		v.StopPatternID.OrSet(0)
		v.JourneyPatternOffset.OrSet(0)
	case *gtfs.Transfer:
		v.TransferType.OrSet(0)
	case *gtfs.Calendar:
		v.Generated.OrSet(false)
		v.Monday.OrSet(0)
		v.Tuesday.OrSet(0)
		v.Wednesday.OrSet(0)
		v.Thursday.OrSet(0)
		v.Friday.OrSet(0)
		v.Saturday.OrSet(0)
		v.Sunday.OrSet(0)
	}
	return nil
}

// emptyTextFilter sets the text fields the database needs non-null to empty
// strings when unset. agency_id and agency_url have not-null constraints, and the
// generated textsearch columns join the rest without COALESCE, so one null would
// leave a row's whole vector null.
//
// An empty string reads back from the database as unset, so readers see no
// change, unlike NotNullFilter's defaults.
type emptyTextFilter struct{}

func (e *emptyTextFilter) Filter(ent tt.Entity, emap *tt.EntityMap) error {
	switch v := ent.(type) {
	case *gtfs.Agency:
		v.AgencyID.OrSet("")
		v.AgencyURL.OrSet("")
		v.AgencyEmail.OrSet("")
	case *gtfs.Route:
		v.RouteLongName.OrSet("")
		v.RouteShortName.OrSet("")
		v.RouteDesc.OrSet("")
	case *gtfs.Stop:
		v.StopName.OrSet("")
		v.StopDesc.OrSet("")
		v.StopCode.OrSet("")
		v.StopURL.OrSet("")
	}
	return nil
}
