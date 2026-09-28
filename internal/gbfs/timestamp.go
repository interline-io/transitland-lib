package gbfs

import (
	"io"

	"github.com/interline-io/transitland-lib/tt"
)

// Timestamp is a time in POSIX seconds, read from a number or an RFC3339
// string (3.x times and 2.3 Datetime fields).
type Timestamp struct {
	tt.Int
}

// UnmarshalJSON leaves an unparseable value unset rather than failing the
// whole file.
func (r *Timestamp) UnmarshalJSON(b []byte) error {
	if r.Int.UnmarshalJSON(b) == nil {
		return nil
	}
	var t tt.Time
	if t.UnmarshalJSON(b) == nil && t.Valid {
		r.Int = tt.NewInt(int(t.Val.Unix()))
	} else {
		r.Int = tt.Int{}
	}
	return nil
}

// MarshalGQL writes POSIX seconds. It is declared here rather than promoted
// from tt.Int, because gqlgen does not bind promoted methods.
func (r Timestamp) MarshalGQL(w io.Writer) { r.Int.MarshalGQL(w) }

// UnmarshalGQL reads POSIX seconds.
func (r *Timestamp) UnmarshalGQL(v any) error { return r.Int.UnmarshalGQL(v) }
