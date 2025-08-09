package clock

import (
	"time"

	"github.com/interline-io/log"
)

// Allow for time mocking
type Clock interface {
	Now() time.Time
}

// Real system clock
type Real struct{}

func (dc *Real) Now() time.Time {
	return time.Now().In(time.UTC)
}

// A mock clock with a fixed time
type Mock struct {
	T time.Time
}

func (dc *Mock) Now() time.Time {
	return dc.T
}

// Helpers

func tzTruncate(s time.Time, loc *time.Location) time.Time {
	if loc == nil {
		log.Error().Msg("tzTruncate: loc is nil, set to UTC")
		loc = time.UTC
	}
	return time.Date(s.Year(), s.Month(), s.Day(), 0, 0, 0, 0, loc)
}
