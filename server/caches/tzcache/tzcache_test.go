package tzcache

import (
	"sync"
	"testing"
	"time"
)

func BenchmarkLoadLocation(b *testing.B) {
	t := "America/Los_Angeles"
	time.LoadLocation(t)
	for n := 0; n < b.N; n++ {
		loc, err := time.LoadLocation(t)
		_ = loc
		_ = err
	}
}

func BenchmarkLoadLocationCache(b *testing.B) {
	lock := sync.Mutex{}
	c := map[string]*time.Location{}
	t := "America/Los_Angeles"
	time.LoadLocation(t)
	for n := 0; n < b.N; n++ {
		lock.Lock()
		if _, ok := c[t]; !ok {
			loc, err := time.LoadLocation(t)
			_ = loc
			_ = err
			c[t] = loc
		}
		lock.Unlock()
	}
}

func BenchmarkCache(b *testing.B) {
	c := NewCache[int]()
	for n := 0; n < b.N; n++ {
		loc, ok := c.Add(n, "America/Los_Angeles")
		_ = loc
		_ = ok
	}
}
