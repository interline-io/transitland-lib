package tt

type Bool struct {
	Option[bool]
}

func NewBool(v bool) Bool {
	return Bool{Option: NewOption(v)}
}
