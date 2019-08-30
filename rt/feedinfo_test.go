package rt

import (
	"testing"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/internal/testutil"
)

func newReader() gotransit.Reader {
	_, r := testutil.NewMinimalTestFeed()
	return r
}

func TestFeedInfo_Contains(t *testing.T) {
	r := newReader()
	fi, err := NewFeedInfoFromReader(r)
	if err != nil {
		t.Error(err)
	}
	if !fi.Contains("agency.txt", "agency1") {
		t.Error("expected")
	}
}
