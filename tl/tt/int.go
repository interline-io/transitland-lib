package tt

import (
	"database/sql/driver"
	"io"
	"strconv"
)

// Int is a nullable int
type Int struct {
	Option[int]
}

func NewInt(v int) Int {
	return Int{Option[int]{Valid: true, Val: v}}
}

func (r *Int) String() string {
	return strconv.Itoa(r.Val)
}

func (r Int) Value() (driver.Value, error) {
	if !r.Valid {
		return nil, nil
	}
	return int64(r.Val), nil
}

// Needed for gqlgen - issue with generics
func (r *Int) UnmarshalGQL(v interface{}) error {
	return r.Scan(v)
}

func (r Int) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
