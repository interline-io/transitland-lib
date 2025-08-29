package httpcache

import (
	"time"

	"github.com/jellydator/ttlcache/v2"
)

type TTLCache struct {
	cache *ttlcache.Cache
}

func NewTTLCache(size int, d time.Duration) *TTLCache {
	c := ttlcache.NewCache()
	c.SetTTL(d)
	c.SetCacheSizeLimit(size)
	return &TTLCache{cache: c}
}

func (c *TTLCache) Get(key string) (interface{}, bool) {
	value, exists := c.cache.Get(key)
	return value, exists == nil
}

func (c *TTLCache) Set(key string, value interface{}) error {
	return c.cache.Set(key, value)
}

func (c *TTLCache) Len() int {
	return c.cache.Count()
}

func (c *TTLCache) Close() error {
	return c.cache.Close()
}

func (c *TTLCache) SkipExtension(ok bool) {
	c.cache.SkipTTLExtensionOnHit(ok)
}
