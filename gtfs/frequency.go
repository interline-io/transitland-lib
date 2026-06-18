package gtfs

import (
	"fmt"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// Frequency frequencies.txt
type Frequency struct {
	TripID      tt.String  `csv:",required" target:"trips.txt" standardized_sort:"1"`
	HeadwaySecs tt.Int     `csv:",required" range:"1,"`
	StartTime   tt.Seconds `csv:",required" standardized_sort:"2"`
	EndTime     tt.Seconds `csv:",required"`
	ExactTimes  tt.Int     `enum:"0,1"`
	tt.BaseEntity
}

// Filename frequencies.txt
func (ent *Frequency) Filename() string {
	return "frequencies.txt"
}

// TableName gtfs_frequencies
func (ent *Frequency) TableName() string {
	return "gtfs_frequencies"
}

// RepeatCount returns the number of times this trip will be repeated.
func (ent *Frequency) RepeatCount() int {
	if ent.HeadwaySecs.Val <= 0 {
		return 0
	}
	count := 0
	for t := ent.StartTime.Int(); t <= ent.EndTime.Int(); t += ent.HeadwaySecs.Int() {
		count++
	}
	return count
}

// Errors for this Entity.
func (ent *Frequency) ConditionalErrors() (errs []error) {
	st, et := ent.StartTime.Int(), ent.EndTime.Int()
	if st != 0 && et != 0 && st > et {
		errs = append(errs, causes.NewInvalidFieldError("end_time", fmt.Sprintf("%d", et), fmt.Errorf("end_time '%d' must come after start_time '%d'", et, st)))
	}
	return errs
}
