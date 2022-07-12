package tt

import "io"

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

// Needed for gqlgen - issue with generics
func (r *String) UnmarshalGQL(v interface{}) error {
	return r.Scan(v)
}

func (r String) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}
