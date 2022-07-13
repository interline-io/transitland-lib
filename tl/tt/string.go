package tt

type String struct {
	Option[string]
}

func NewString(v string) String {
	return String{Option[string]{Valid: true, Val: v}}
}
