package gtfs

import (
	"fmt"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// Frequency frequencies.txt
type Frequency struct {
	TripID tt.String `csv:",required" target:"trips.txt" standardized_sort:"1"`
	// Stricter than GTFS, which allows any positive headway. A headway under a
	// minute is almost always a data error, and expands into a flood of departures.
	HeadwaySecs tt.Int     `csv:",required" range:"60,"`
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

// Starts returns the times this frequency's trips leave the first stop: every
// headway_secs from start_time, up to but not including end_time.
func (ent *Frequency) Starts() []int {
	if ent.HeadwaySecs.Val <= 0 {
		return nil
	}
	var ret []int
	for t := ent.StartTime.Int(); t < ent.EndTime.Int(); t += ent.HeadwaySecs.Int() {
		ret = append(ret, t)
	}
	return ret
}

// RepeatCount returns the number of trips this frequency runs.
func (ent *Frequency) RepeatCount() int {
	return len(ent.Starts())
}

// Errors for this Entity.
func (ent *Frequency) ConditionalErrors() (errs []error) {
	// A period that ends at or before its start holds no runs, so the row is
	// rejected and its trip runs at its own stop times instead. Missing times are
	// reported by the required check; 00:00:00 is a valid time, not a missing one.
	st, et := ent.StartTime.Int(), ent.EndTime.Int()
	if ent.StartTime.Valid && ent.EndTime.Valid && et <= st {
		errs = append(errs, causes.NewInvalidFieldError("end_time", fmt.Sprintf("%d", et), fmt.Errorf("end_time '%d' must come after start_time '%d'", et, st)))
	}
	return errs
}
