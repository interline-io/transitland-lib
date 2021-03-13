package tl

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl/causes"
)

// Frequency frequencies.txt
type Frequency struct {
	TripID      string   `csv:"trip_id,required"`
	StartTime   WideTime `csv:"start_time,required"`
	EndTime     WideTime `csv:"end_time,required"`
	HeadwaySecs int      `csv:"headway_secs,required"`
	ExactTimes  int
	BaseEntity
}

// RepeatCount returns the number of times this trip will be repeated.
func (ent *Frequency) RepeatCount() int {
	if ent.HeadwaySecs <= 0 {
		return 0
	}
	count := 0
	for t := ent.StartTime.Seconds; t <= ent.EndTime.Seconds; t += ent.HeadwaySecs {
		count++
	}
	return count
}

// Errors for this Entity.
func (ent *Frequency) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.loadErrors...)
	st, et := ent.StartTime.Seconds, ent.EndTime.Seconds
	if ent.HeadwaySecs < 1 {
		errs = append(errs, causes.NewInvalidFieldError("headway_secs", "", fmt.Errorf("headway_secs must be a positive integer")))
	}
	if st != 0 && et != 0 && st > et {
		errs = append(errs, causes.NewInvalidFieldError("end_time", "", fmt.Errorf("end_time '%d' must come after start_time '%d'", et, st)))
	}
	if !(ent.ExactTimes == 0 || ent.ExactTimes == 1) {
		errs = append(errs, causes.NewInvalidFieldError("exact_times", "", fmt.Errorf("exact_times must be 0 or 1")))
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
	if tripID, ok := emap.GetEntity(&Trip{TripID: ent.TripID}); ok {
		ent.TripID = tripID
	} else {
		return causes.NewInvalidReferenceError("trip_id", ent.TripID)
	}
	return nil
}
