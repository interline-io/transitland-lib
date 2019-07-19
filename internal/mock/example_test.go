package mock

import (
	"testing"
)

func TestExampleReader(t *testing.T) {
	r := NewExampleExpect()
	TestExpect(t, *r, r.Reader)
}

func TestExampleWriter(t *testing.T) {
	r := NewExampleExpect()
	writer := Writer{}
	DirectCopy(r.Reader, &writer)
	r2, err := writer.NewReader()
	if err != nil {
		t.Error(err)
	}
	TestExpect(t, *r, r2)
}
