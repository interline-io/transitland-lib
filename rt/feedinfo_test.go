package rt

import (
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tl"
)

func newReader() tl.Reader {
	_, r := testutil.NewMinimalTestFeed()
	return r
}

// func TestFeedInfo_Contains(t *testing.T) {
// 	r := newReader()
// 	fi, err := NewFeedInfoFromReader(r)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	if !fi.Contains("agency.txt", "agency1") {
// 		t.Error("expected")
// 	}
// }
