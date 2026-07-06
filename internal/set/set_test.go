package set

import (
	"sort"
	"testing"
)

func TestSet(t *testing.T) {
	s := New(1, 2, 2, 3)
	if s.Cardinality() != 3 {
		t.Fatalf("Cardinality = %d, want 3", s.Cardinality())
	}
	if !s.Contains(2) || s.Contains(9) {
		t.Fatal("Contains mismatch")
	}
	s.Add(4)
	s.Add(4)
	if s.Cardinality() != 4 {
		t.Fatalf("Cardinality after Add = %d, want 4", s.Cardinality())
	}

	a := New(1, 2, 3)
	b := New(2, 3, 4)
	if got := sortedSlice(a.Intersect(b)); !equal(got, []int{2, 3}) {
		t.Fatalf("Intersect = %v, want [2 3]", got)
	}
	if got := sortedSlice(a.Difference(b)); !equal(got, []int{1}) {
		t.Fatalf("Difference = %v, want [1]", got)
	}

	seen := New[int]()
	for v := range a.Iter() {
		seen.Add(v)
	}
	if !equal(sortedSlice(seen), []int{1, 2, 3}) {
		t.Fatalf("Iter visited %v, want [1 2 3]", sortedSlice(seen))
	}
}

func sortedSlice(s Set[int]) []int {
	out := s.ToSlice()
	sort.Ints(out)
	return out
}

func equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
