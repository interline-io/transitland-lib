package testutil

// CompareSliceString compares two slices of strings
func CompareSliceString(a []string, b []string) bool {
	am := map[string]bool{}
	for _, i := range a {
		am[i] = true
	}
	bm := map[string]bool{}
	for _, i := range b {
		bm[i] = true
	}
	if len(am) != len(bm) {
		return false
	}
	for k, v := range am {
		if bm[k] != v {
			return false
		}
	}
	return true
}

// CompareSliceInt compares two slices of ints
func CompareSliceInt(a []int, b []int) bool {
	am := map[int]bool{}
	for _, i := range a {
		am[i] = true
	}
	bm := map[int]bool{}
	for _, i := range b {
		bm[i] = true
	}
	if len(am) != len(bm) {
		return false
	}
	for k, v := range am {
		if bm[k] != v {
			return false
		}
	}
	return true
}
