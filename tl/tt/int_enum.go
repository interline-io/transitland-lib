package tt

import (
	"database/sql/driver"
	"strconv"
)

// IntEnum is an integer enum, commonly used in GTFS
type IntEnum struct {
	Option[int]
}

func NewIntEnum(v int) IntEnum {
	return IntEnum{Option[int]{Valid: true, Val: v}}
}

func (r IntEnum) String() string {
	return strconv.Itoa(r.Val)
}

func (r IntEnum) Value() (driver.Value, error) {
	if !r.Valid {
		return nil, nil
	}
	return int64(r.Val), nil
}
