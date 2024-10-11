package tt

import "strconv"

// Key is a nullable foreign key constraint, similar to sql.NullString
type Key struct {
	Option[string]
}

func NewKey(v string) Key {
	return Key{Option: NewOption(v)}
}

func (r Key) Int() int {
	a, _ := strconv.Atoi(r.Val)
	return a
}
