package tt

import (
	"fmt"
	"io"
)

// Float is a nullable float
type Float struct {
	Option[float64]
}

func NewFloat(v float64) Float {
	return Float{Option[float64]{Valid: true, Val: v}}
}

func (r Float) String() string {
	if !r.Valid {
		return ""
	}
	if r.Val > -100_000 && r.Val < 100_000 {
		return fmt.Sprintf("%g", r.Val)
	}
	return fmt.Sprintf("%0.5f", r.Val)
}

func (r *Float) Float64() float64 {
	return r.Val
}

// Needed for gqlgen - issue with generics
func (r *Float) UnmarshalGQL(v interface{}) error {
	return r.Scan(v)
}

func (r Float) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
