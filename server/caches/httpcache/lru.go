package httpcache

import "github.com/tidwall/tinylru"

type LRUCache struct {
	*tinylru.LRU
}

func NewLRUCache(size int) *LRUCache {
	c := tinylru.LRU{}
	c.Resize(size)
	return &LRUCache{LRU: &c}
}

func (c *LRUCache) Get(key string) (interface{}, bool) {
	return c.LRU.Get(key)
}

func (c *LRUCache) Set(key string, value interface{}) error {
	c.LRU.Set(key, value)
	return nil
}

func (c *LRUCache) Len() int {
	return c.LRU.Len()
}

func (c *LRUCache) Close() error {
	return nil
}
