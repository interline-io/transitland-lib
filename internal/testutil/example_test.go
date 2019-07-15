package testutil

import (
	"fmt"
	"testing"

	"github.com/interline-io/gotransit/copier"
)

func TestMockReader(t *testing.T) {
	r := ExampleReader()
	writer := MockWriter{}
	cp := copier.NewCopier(r, &writer)
	err := cp.Copy()
	fmt.Printf("err: %#v\n", err)
}
