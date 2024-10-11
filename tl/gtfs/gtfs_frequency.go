package gtfs

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tt"
)

// Frequency frequencies.txt
type Frequency struct {
	TripID      string `csv:",required"`
	HeadwaySecs int    `csv:",required"`
	StartTime   tt.Seconds
	EndTime     tt.Seconds
	ExactTimes  int
	BaseEntity
}

// RepeatCount returns the number of times this trip will be repeated.
func (ent *Frequency) RepeatCount() int {
	if ent.HeadwaySecs <= 0 {
		return 0
	}
	count := 0
	for t := ent.StartTime.Int(); t <= ent.EndTime.Int(); t += ent.HeadwaySecs {
		count++
	}
	return count
}

// Errors for this Entity.
func (ent *Frequency) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.loadErrors...)
	if !ent.StartTime.Valid {
		errs = append(errs, causes.NewRequiredFieldError("start_time"))
	}
	if !ent.EndTime.Valid {
		errs = append(errs, causes.NewRequiredFieldError("end_time"))
	}
	st, et := ent.StartTime.Int(), ent.EndTime.Int()
	if ent.HeadwaySecs < 1 {
		errs = append(errs, causes.NewInvalidFieldError("headway_secs", fmt.Sprintf("%d", ent.HeadwaySecs), fmt.Errorf("headway_secs must be a positive integer")))
	}
	if st != 0 && et != 0 && st > et {
		errs = append(errs, causes.NewInvalidFieldError("end_time", fmt.Sprintf("%d", et), fmt.Errorf("end_time '%d' must come after start_time '%d'", et, st)))
	}
	if !(ent.ExactTimes == 0 || ent.ExactTimes == 1) {
		errs = append(errs, causes.NewInvalidFieldError("exact_times", fmt.Sprintf("%d", ent.ExactTimes), fmt.Errorf("exact_times must be 0 or 1")))
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
