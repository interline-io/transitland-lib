package tt

import "strconv"

type String struct {
	Option[string]
}

func NewString(v string) String {
	return String{Option: NewOption(v)}
}

// TODO: Consider restricting valid to non-empty strings
// func (r *String) Set(v string) {
// 	r.Val = v
// 	r.Valid = r.Val != ""
// }

func (r String) String() string {
	return r.Val
}

func (r String) Int() int {
	if !r.Valid {
		return 0
	}
	a, _ := strconv.ParseInt(r.Val, 10, 64)
	return int(a)
}

func (r String) IsPresent() bool {
	return r.Valid && r.Val != ""
}
