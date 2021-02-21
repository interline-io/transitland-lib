package plus

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
)

// RealtimeTrip realtime_trips.txt
type RealtimeTrip struct {
	TripID         string `csv:"trip_id"`
	RealtimeTripID string `csv:"realtime_trip_id"`
	tl.BaseEntity
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
func (ent *RealtimeTrip) UpdateKeys(emap *tl.EntityMap) error {
	if fkid, ok := emap.GetEntity(&tl.Trip{TripID: ent.TripID}); ok {
		ent.TripID = fkid
	} else {
		return causes.NewInvalidReferenceError("trip_id", ent.TripID)
	}
	return nil
}
