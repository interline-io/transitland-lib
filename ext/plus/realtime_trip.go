package plus

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// RealtimeTrip realtime_trips.txt
type RealtimeTrip struct {
	TripID         string `csv:"trip_id"`
	RealtimeTripID string `csv:"realtime_trip_id"`
	tt.BaseEntity
}

// Filename realtime_trips.txt
func (ent *RealtimeTrip) Filename() string {
	return "realtime_trips.txt"
}

// TableName ext_plus_realtime_trips
func (ent *RealtimeTrip) TableName() string {
	return "ext_plus_realtime_trips"
}

// UpdateKeys updates Entity references.
func (ent *RealtimeTrip) UpdateKeys(emap *tt.EntityMap) error {
	if fkid, ok := emap.GetEntity(&gtfs.Trip{TripID: ent.TripID}); ok {
		ent.TripID = fkid
	} else {
		return causes.NewInvalidReferenceError("trip_id", ent.TripID)
	}
	return nil
}
