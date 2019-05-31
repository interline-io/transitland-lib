package rt

import "testing"

func Test_readmsg(t *testing.T) {
	err := readmsg("../testdata/rt/example.pbf")
	if err != nil {
		t.Error(err)
	}
}
