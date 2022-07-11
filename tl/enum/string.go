package enum

// String is a nullable string, with additional methods for gql and json.
type String struct {
	Option[string]
}

func NewString(v string) String {
	return String{Option[string]{Valid: (v != ""), Val: v}}
}
