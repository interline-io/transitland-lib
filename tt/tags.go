package tt

// Tags is a map[string]string with json and gql marshal support.
// This is a struct instead of bare map[string]string because of a gqlgen issue.
type Tags struct {
	Option[map[string]string]
}

// Keys return the tag keys
func (r Tags) Keys() []string {
	var ret []string
	for k := range r.Val {
		ret = append(ret, k)
	}
	return ret
}

// Set a tag value
func (r *Tags) Set(k, v string) {
	if r.Val == nil {
		r.Val = map[string]string{}
	}
	r.Val[k] = v
}

// Get a tag value by key
func (r Tags) Get(k string) (string, bool) {
	if r.Val == nil {
		return "", false
	}
	a, ok := r.Val[k]
	return a, ok
}
