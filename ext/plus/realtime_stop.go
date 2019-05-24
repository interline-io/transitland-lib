package plus

import (
	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/causes"
)

// RealtimeStop realtime_stops.txt
type RealtimeStop struct {
	TripID         string `csv:"trip_id" gorm:"type:int"`
	StopID         string `csv:"stop_id" gorm:"type:int"`
	RealtimeStopID string `csv:"realtime_stop_id"`
	gotransit.BaseEntity
}

// Filename realtime_stops.txt
func (ent *RealtimeStop) Filename() string {
	return "realtime_stops.txt"
}

// TableName ext_plus_realtime_stops
func (ent *RealtimeStop) TableName() string {
	return "ext_plus_realtime_stops"
}

// UpdateKeys updates Entity references.
func (ent *RealtimeStop) UpdateKeys(emap *gotransit.EntityMap) error {
	if fkid, ok := emap.Get(&gotransit.Trip{TripID: ent.TripID}); ok {
		ent.TripID = fkid
	} else {
		return causes.NewInvalidReferenceError("trip_id", ent.TripID)
	}
	if fkid, ok := emap.Get(&gotransit.Stop{StopID: ent.StopID}); ok {
		ent.StopID = fkid
	} else {
		return causes.NewInvalidReferenceError("stop_id", ent.StopID)
	}
	// if fkid, ok := emap.Get(&gotransit.Stop{StopID: ent.RealtimeStopID}); ok {
	// 	ent.RealtimeStopID = fkid
	// } else {
	// 	return causes.NewInvalidReferenceError("stop_id", ent.StopID)
	// }
	return nil
}
