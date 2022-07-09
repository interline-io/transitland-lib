package enum

import (
	"database/sql/driver"
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
