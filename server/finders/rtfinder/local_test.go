package rtfinder

import (
	"testing"
)

func TestLocalCache(t *testing.T) {
	rtCache := NewLocalCache()
	testCache(t, rtCache)
}
