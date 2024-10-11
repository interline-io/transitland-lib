package filters

import (
	"encoding/json"
	"fmt"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tt"
)

func NewApplyTimezoneFilter(timezone string) (*ApplyTimezoneFilter, error) {
	if _, ok := tt.IsValidTimezone(timezone); !ok {
		return nil, fmt.Errorf("invalid timezone '%s'", timezone)
	}
	if timezone == "" {
		return nil, fmt.Errorf("a timezone must be specified")
	}
	return &ApplyTimezoneFilter{
		timezone: timezone,
	}, nil
}

func newApplyTimezoneFilterFromJson(args string) (*ApplyTimezoneFilter, error) {
	type tzOptions struct {
		Timezone string
	}
	opts := &tzOptions{}
	if err := json.Unmarshal([]byte(args), opts); err != nil {
		return nil, err
	}
	return NewApplyTimezoneFilter(opts.Timezone)
}

// ApplyTimezoneFilter sets all timezones in the file to the specified value
type ApplyTimezoneFilter struct {
	timezone string
}

func (e *ApplyTimezoneFilter) Filter(ent tl.Entity, emap *tl.EntityMap) error {
	switch v := ent.(type) {
	case *tl.Agency:
		v.AgencyTimezone = e.timezone
	case *tl.Stop:
		v.StopTimezone = e.timezone
	}
	return nil
}
