package tt

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
