package rtfinder

import (
	"testing"

	"github.com/interline-io/transitland-mw/testutil"
)

func TestRedisCache(t *testing.T) {
	// redis jobs and cache
	if a, ok := testutil.CheckTestRedisClient(); !ok {
		t.Skip(a)
		return
	}
	client := testutil.MustOpenTestRedisClient(t)
	rtCache := NewRedisCache(client)
	testCache(t, rtCache)
}
