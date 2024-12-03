package tt

import (
	"fmt"
)

// Float is a nullable float64
type Float struct {
	Option[float64]
}

func NewFloat(v float64) Float {
	return Float{Option: NewOption(v)}
}

func (r Float) String() string {
	if r.Val > -100_000 && r.Val < 100_000 {
		return fmt.Sprintf("%g", r.Val)
	}
	return fmt.Sprintf("%0.5f", r.Val)
}

func (r Float) Float() float64 {
	return float64(r.Val)
}
