package tt

// String is a nullable string, with additional methods for gql and json.
// This could be converted to Option[string]
type String struct {
	Option[string]
}

func NewString(v string) String {
	return String{Option[string]{Valid: (v != ""), Val: v}}
}

func (r *String) Present() bool {
	if r.Val != "" {
		return true
	}
	return false
}

func (r *String) Error() error {
	return nil
}
