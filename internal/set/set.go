// Package set provides a minimal generic set backed by a map.
package set

import "iter"

// Set is an unordered collection of unique values. It is not safe for concurrent use.
type Set[T comparable] map[T]struct{}

// New returns a set containing the given items.
func New[T comparable](items ...T) Set[T] {
	s := make(Set[T], len(items))
	for _, item := range items {
		s[item] = struct{}{}
	}
	return s
}

// Add inserts an item into the set.
func (s Set[T]) Add(item T) {
	s[item] = struct{}{}
}

// Contains reports whether the set contains item.
func (s Set[T]) Contains(item T) bool {
	_, ok := s[item]
	return ok
}

// Cardinality returns the number of items in the set.
func (s Set[T]) Cardinality() int {
	return len(s)
}

// ToSlice returns the set's items in unspecified order.
func (s Set[T]) ToSlice() []T {
	items := make([]T, 0, len(s))
	for item := range s {
		items = append(items, item)
	}
	return items
}

// Intersect returns a new set of the items present in both s and other.
func (s Set[T]) Intersect(other Set[T]) Set[T] {
	// Iterate the smaller set to minimize lookups.
	a, b := s, other
	if len(b) < len(a) {
		a, b = b, a
	}
	out := make(Set[T])
	for item := range a {
		if _, ok := b[item]; ok {
			out[item] = struct{}{}
		}
	}
	return out
}

// Difference returns a new set of the items in s that are not in other.
func (s Set[T]) Difference(other Set[T]) Set[T] {
	out := make(Set[T])
	for item := range s {
		if _, ok := other[item]; !ok {
			out[item] = struct{}{}
		}
	}
	return out
}

// Iter iterates over the set's items in unspecified order.
func (s Set[T]) Iter() iter.Seq[T] {
	return func(yield func(T) bool) {
		for item := range s {
			if !yield(item) {
				return
			}
		}
	}
}
