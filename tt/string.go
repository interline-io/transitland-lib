package tt

import "strconv"

type String struct {
	Option[string]
}

func NewString(v string) String {
	return String{Option: NewOption(v)}
}

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
