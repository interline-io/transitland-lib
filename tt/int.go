package tt

import "database/sql/driver"

// Int is a nullable int
type Int struct {
	Option[int64]
}

func NewInt(v int) Int {
	return Int{Option: NewOption(int64(v))}
}

func (r *Int) SetInt(v int) {
	r.Val = int64(v)
	r.Valid = true
}

// Int is a convenience function for int(v)
func (r Int) Int() int {
	return int(r.Val)
}

func (r Int) Float() float64 {
	return float64(r.Val)
}

///////////

// Same as int but writes 0 to database
// Use when a field is OPTIONAL in spec, but we need to mainain a NOT NULL for query purposes
type DefaultInt struct {
	Option[int64]
}

func (r DefaultInt) Value() (driver.Value, error) {
	return r.Val, nil
}

func (r DefaultInt) Int() int {
	return int(r.Val)
}

func (r *DefaultInt) SetInt(v int) {
	r.Val = int64(v)
}

func (r DefaultInt) Float() float64 {
	return float64(r.Val)
}
