package tt

// Ints is a nullable slice of []int
type Ints struct {
	Option[[]int64]
}

func NewInts(v []int) Ints {
	x := make([]int64, len(v))
	for i := range v {
		x[i] = int64(v[i])
	}
	return Ints{Option: NewOption(x)}
}
