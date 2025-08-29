package httpcache

import (
	"net/http"
)

type Cacher interface {
	Get(string) (interface{}, bool)
	Set(string, interface{}) error
	Len() int
	Close() error
}

type Cache struct {
	key          HTTPKey
	roundTripper http.RoundTripper
	cacher       Cacher
}

func NewCache(rt http.RoundTripper, key HTTPKey, cacher Cacher) *Cache {
	if key == nil {
		key = DefaultKey
	}
	if rt == nil {
		rt = http.DefaultTransport
	}
	if cacher == nil {
		cacher = NewLRUCache(16 * 1024)
	}
	return &Cache{
		roundTripper: rt,
		key:          key,
		cacher:       cacher,
	}
}

func (h *Cache) makeRequest(req *http.Request, key string) (*http.Response, error) {
	// Make request
	res, err := h.roundTripper.RoundTrip(req)
	if err != nil {
		return res, err
	}
	// Save response
	rr, err := newCacheResponse(res)
	if err != nil {
		return nil, err
	}
	h.cacher.Set(key, rr)
	return res, nil
}

func (h *Cache) check(key string) (*http.Response, error) {
	if a, ok := h.cacher.Get(key); ok {
		v, ok := a.(*cacheResponse)
		if ok {
			return fromCacheResponse(v)
		}
	}
	return nil, nil
}

func (h *Cache) RoundTrip(req *http.Request) (*http.Response, error) {
	key := h.key(req)
	if a, err := h.check(key); a != nil {
		// fmt.Println("Cache: got cached:", key)
		return a, err
	}
	rr, err := h.makeRequest(req, key)
	// fmt.Println("Cache: saved to cache:", key)
	return rr, err
}
