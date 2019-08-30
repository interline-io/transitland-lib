package gotransit

import (
	"fmt"

	"github.com/interline-io/gotransit/causes"
)

// Frequency frequencies.txt
type Frequency struct {
	TripID      string   `csv:"trip_id" required:"true"`
	StartTime   WideTime `csv:"start_time" required:"true"`
	EndTime     WideTime `csv:"end_time" required:"true"`
	HeadwaySecs int      `csv:"headway_secs" min:"1" required:"true"`
	ExactTimes  int      `csv:"exact_times" min:"0" max:"1"`
	BaseEntity
}

// EntityID returns nothing.
func (ent *Frequency) EntityID() string {
	return ""
}

// Warnings for this Entity.
func (ent *Frequency) Warnings() (errs []error) {
	st, et := ent.StartTime.Seconds, ent.EndTime.Seconds
	if st != 0 && et != 0 {
		if st == et {
			errs = append(errs, causes.NewValidationWarning("end_time", "end_time is equal to start_time"))
		}
		if (et - st) < ent.HeadwaySecs {
			errs = append(errs, causes.NewValidationWarning("end_time", "end_time is less than start_time + headway_secs"))
		}
	}
	return errs
}

// Errors for this Entity.
func (ent *Frequency) Errors() (errs []error) {
	errs = ValidateTags(ent)
	errs = append(errs, ent.BaseEntity.loadErrors...)
	st, et := ent.StartTime.Seconds, ent.EndTime.Seconds
	if st != 0 && et != 0 && st > et {
		errs = append(errs, causes.NewInvalidFieldError("end_time", "", fmt.Errorf("end_time '%d' must come after start_time '%d'", et, st)))
	}
	return errs
}

// Filename frequencies.txt
func (ent *Frequency) Filename() string {
	return "frequencies.txt"
}

// TableName gtfs_frequencies
func (ent *Frequency) TableName() string {
	return "gtfs_frequencies"
}

// UpdateKeys updates Entity references.
func (ent *Frequency) UpdateKeys(emap *EntityMap) error {
	// Adjust TripID
	if tripID, ok := emap.Get(&Trip{TripID: ent.TripID}); ok {
		ent.TripID = tripID
	} else {
		return causes.NewInvalidReferenceError("trip_id", ent.TripID)
	}
	return nil
}
