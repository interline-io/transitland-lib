package tt

import "strconv"

// Key is a nullable foreign key constraint, similar to sql.NullString
type Key struct {
	Option[string]
}

func (r *Key) Set(v string) {
	r.Val = v
	r.Valid = r.Val != ""
}

func (r *Key) SetInt(v int) {
	r.Val = strconv.Itoa(v)
	r.Valid = true
}

func NewKey(v string) Key {
	return Key{Option: NewOption(v)}
}

func (r Key) IsPresent() bool {
	return r.Valid && r.Val != ""
}

func (r Key) Int() int {
	a, _ := strconv.Atoi(r.Val)
	return a
}
