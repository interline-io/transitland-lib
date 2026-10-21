package clock

import "time"

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
