package tldb

import (
	"strconv"
	"testing"
)

// Reader tests are handled as part of the round-trip Writer tests.

func Test_chunkStrings(t *testing.T) {
	bsize := 95
	csize := 10
	var a []string
	for i := 0; i < bsize; i++ {
		a = append(a, strconv.Itoa(i))
	}
	c := chunkStrings(a, csize)
	// fmt.Println(a)
	// fmt.Println(c)
	if len(a) != bsize {
		t.Errorf("got input size %d, expected %d", len(a), bsize)
	}
	es := (bsize / csize)
	if bsize%csize > 0 {
		es += 1
	}
	if len(c) != es {
		t.Errorf("got output size %d, expected %d", len(c), es)
	}
	ec := 0
	for _, v := range c {
		ec += len(v)
	}
	if ec != bsize {
		t.Errorf("got ouput length %d, expected %d", ec, bsize)
	}
}
