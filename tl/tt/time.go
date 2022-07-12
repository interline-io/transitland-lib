package tt

import (
	"io"
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

// Needed for gqlgen - issue with generics
func (r *Time) UnmarshalGQL(v interface{}) error {
	return r.Scan(v)
}

func (r Time) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
