package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// FrequencyOverlapError is reported when two frequencies.txt entries for the same trip overlap in time.
type FrequencyOverlapError struct {
	StartTime      tt.Seconds
	EndTime        tt.Seconds
	OtherStartTime tt.Seconds
	OtherEndTime   tt.Seconds
	TripID         string
	bc
}

func (e *FrequencyOverlapError) Error() string {
	return fmt.Sprintf(
		"frequency block for trip '%s' with interval %s -> %s overlaps another frequency block for this trip with interval %s -> %s",
		e.TripID,
		e.StartTime.String(),
		e.EndTime.String(),
		e.OtherStartTime.String(),
		e.OtherEndTime.String(),
	)
}

type freqValue struct {
	start int
	end   int
}

// FrequencyOverlapCheck checks for FrequencyOverlapErrors.
type FrequencyOverlapCheck struct {
	freqs map[string][]*freqValue
}

// Validate .
func (e *FrequencyOverlapCheck) Validate(ent tt.Entity) []error {
	v, ok := ent.(*gtfs.Frequency)
	if !ok {
		return nil
	}
	if e.freqs == nil {
		e.freqs = map[string][]*freqValue{}
	}
	var errs []error
	tf := freqValue{
		start: v.StartTime.Int(),
		end:   v.EndTime.Int(),
	}
	for _, hit := range e.freqs[v.TripID] {
		if !(tf.start >= hit.end || tf.end <= hit.start) {
			errs = append(errs, &FrequencyOverlapError{
				TripID:         v.TripID,
				StartTime:      v.StartTime,
				EndTime:        v.EndTime,
				OtherStartTime: tt.NewSeconds(hit.start),
				OtherEndTime:   tt.NewSeconds(hit.end),
			})
		}
	}
	e.freqs[v.TripID] = append(e.freqs[v.TripID], &tf)
	return errs
}
