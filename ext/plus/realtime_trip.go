package plus

import (
	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/causes"
)

// RealtimeTrip realtime_trips.txt
type RealtimeTrip struct {
	TripID         string `csv:"trip_id" gorm:"type:int"`
	RealtimeTripID string `csv:"realtime_trip_id"`
	gotransit.BaseEntity
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
func (ent *RealtimeTrip) UpdateKeys(emap *gotransit.EntityMap) error {
	if fkid, ok := emap.Get(&gotransit.Trip{TripID: ent.TripID}); ok {
		ent.TripID = fkid
	} else {
		return causes.NewInvalidReferenceError("trip_id", ent.TripID)
	}
	return nil
}
