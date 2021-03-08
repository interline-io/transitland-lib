package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl"
)

// FrequencyOverlapError is reported when two frequencies.txt entries for the same trip overlap in time.
type FrequencyOverlapError struct {
	StartTime      tl.WideTime
	EndTime        tl.WideTime
	OtherStartTime tl.WideTime
	OtherEndTime   tl.WideTime
	TripID         string
	bc
}

func (e *FrequencyOverlapError) Error() string {
	return fmt.Sprintf("frequency with start_time %s and end_time %s overlaps with another frequency block for trip '%s' with start_time %s and end_time %s", e.StartTime.String(), e.EndTime.String(), e.TripID, e.OtherStartTime.String(), e.OtherEndTime.String())
}

type freqValue struct {
	start int
	end   int
}

// FrequencyOverlapCheck checks that frequencies for the same trip do not overlap.
type FrequencyOverlapCheck struct {
	freqs map[string][]*freqValue
}

// Validate .
func (e *FrequencyOverlapCheck) Validate(ent tl.Entity) []error {
	v, ok := ent.(*tl.Frequency)
	if !ok {
		return nil
	}
	if e.freqs == nil {
		e.freqs = map[string][]*freqValue{}
	}
	var errs []error
	tf := freqValue{
		start: v.StartTime.Seconds,
		end:   v.EndTime.Seconds,
	}
	for _, hit := range e.freqs[v.TripID] {
		if !(tf.start >= hit.end || tf.end <= hit.start) {
			errs = append(errs, &FrequencyOverlapError{
				StartTime:      v.StartTime,
				EndTime:        v.EndTime,
				TripID:         v.TripID,
				OtherStartTime: tl.NewWideTimeFromSeconds(tf.start),
				OtherEndTime:   tl.NewWideTimeFromSeconds(tf.end),
			})
		}
	}
	e.freqs[v.TripID] = append(e.freqs[v.TripID], &tf)
	return errs
}
