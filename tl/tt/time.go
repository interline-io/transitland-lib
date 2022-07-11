package tt

import (
	"time"
)

// Time is a nullable date/time
type Time struct {
	Option[time.Time]
}

func NewTime(v time.Time) Time {
	return Time{Option[time.Time]{Valid: true, Val: v}}
}

func (r Time) String() string {
	if !r.Valid {
		return ""
	}
	return r.Val.Format(time.RFC3339)
}
