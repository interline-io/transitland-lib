package set

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSet(t *testing.T) {
	s := New(1, 2, 2, 3)
	assert.Equal(t, 3, s.Cardinality())
	assert.True(t, s.Contains(2))
	assert.False(t, s.Contains(9))
	s.Add(4)
	s.Add(4)
	assert.Equal(t, 4, s.Cardinality())

	a := New(1, 2, 3)
	b := New(2, 3, 4)
	assert.ElementsMatch(t, []int{2, 3}, a.Intersect(b).ToSlice())
	assert.ElementsMatch(t, []int{1}, a.Difference(b).ToSlice())

	var seen []int
	for v := range a.Iter() {
		seen = append(seen, v)
	}
	assert.ElementsMatch(t, []int{1, 2, 3}, seen)
}
