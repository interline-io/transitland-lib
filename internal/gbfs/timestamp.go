package gbfs

import (
	"encoding/json"
	"io"
	"time"

	"github.com/interline-io/transitland-lib/tt"
)

// Timestamp is a time in POSIX seconds. 1.x/2.x publish it as a number, and
// 3.x and Datetime fields as RFC3339.
type Timestamp struct {
	tt.Int
}

// UnmarshalJSON accepts POSIX seconds or an RFC3339 string. Anything else is
// left unset rather than failing the whole file.
func (r *Timestamp) UnmarshalJSON(b []byte) error {
	var n json.Number
	var s string
	if json.Unmarshal(b, &n) == nil {
		if v, err := n.Float64(); err == nil {
			r.Int = tt.NewInt(int(v))
		}
	} else if json.Unmarshal(b, &s) == nil {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			r.Int = tt.NewInt(int(t.Unix()))
		}
	}
	return nil
}

// MarshalGQL writes POSIX seconds. It is declared here rather than promoted
// from tt.Int, because gqlgen does not bind promoted methods.
func (r Timestamp) MarshalGQL(w io.Writer) { r.Int.MarshalGQL(w) }

// UnmarshalGQL reads POSIX seconds.
func (r *Timestamp) UnmarshalGQL(v any) error { return r.Int.UnmarshalGQL(v) }
