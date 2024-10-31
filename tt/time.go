package tt

import (
	"time"
)

// Time is a nullable date without time component
type Time struct {
	Option[time.Time]
}

func NewTime(v time.Time) Time {
	return Time{Option: NewOption(v)}
}
