package kvcache_test

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/caches/kvcache"
	"github.com/interline-io/transitland-lib/server/caches/kvcache/storetest"
)

func TestMemoryStore(t *testing.T) {
	storetest.Run(t, func(t *testing.T) kvcache.Store {
		return kvcache.NewMemoryStore()
	})
}
