package gql

import (
	"testing"
)

func TestLevelResolver(t *testing.T) {
	c, _ := newTestClient(t)
	testcases := []testcase{
		// TODO: level by stop
		// TODO: stops by level
	}
	queryTestcases(t, c, testcases)
}
