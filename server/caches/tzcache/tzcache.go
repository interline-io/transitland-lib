package tzcache

import (
	"sync"
	"time"
)

// Cache saves and manages the timezone location cache
type Cache[K comparable] struct {
	lock   sync.Mutex
	tzs    map[string]*time.Location
	values map[K]string
}

func NewCache[K comparable]() *Cache[K] {
	return &Cache[K]{
		tzs:    map[string]*time.Location{},
		values: map[K]string{},
	}
}

func (c *Cache[K]) Get(key K) (*time.Location, bool) {
	defer c.lock.Unlock()
	c.lock.Lock()
	tz, ok := c.values[key]
	if ok {
		return c.load(tz)
	}
	return nil, false
}

func (c *Cache[K]) Location(tz string) (*time.Location, bool) {
	defer c.lock.Unlock()
	c.lock.Lock()
	return c.load(tz)
}
func (c *Cache[K]) Add(key K, tz string) (*time.Location, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.values[key] = tz
	return c.load(tz)
}

func (c *Cache[K]) load(tz string) (*time.Location, bool) {
	loc, ok := c.tzs[tz]
	if !ok {
		var err error
		ok = true
		loc, err = time.LoadLocation(tz)
		if err != nil {
			ok = false
			loc = nil
		}
		c.tzs[tz] = loc
	}
	return loc, ok
}
