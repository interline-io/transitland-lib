// Code generated by github.com/vektah/dataloaden, DO NOT EDIT.

package dataloader

import (
	"sync"
	"time"

	"github.com/interline-io/transitland-lib/server/model"
)

// FeedVersionLoaderConfig captures the config to create a new FeedVersionLoader
type FeedVersionLoaderConfig struct {
	// Fetch is a method that provides the data for the loader
	Fetch func(keys []int) ([]*model.FeedVersion, []error)

	// Wait is how long wait before sending a batch
	Wait time.Duration

	// MaxBatch will limit the maximum number of keys to send in one batch, 0 = not limit
	MaxBatch int
}

// NewFeedVersionLoader creates a new FeedVersionLoader given a fetch, wait, and maxBatch
func NewFeedVersionLoader(config FeedVersionLoaderConfig) *FeedVersionLoader {
	return &FeedVersionLoader{
		fetch:    config.Fetch,
		wait:     config.Wait,
		maxBatch: config.MaxBatch,
	}
}

// FeedVersionLoader batches and caches requests
type FeedVersionLoader struct {
	// this method provides the data for the loader
	fetch func(keys []int) ([]*model.FeedVersion, []error)

	// how long to done before sending a batch
	wait time.Duration

	// this will limit the maximum number of keys to send in one batch, 0 = no limit
	maxBatch int

	// INTERNAL

	// lazily created cache
	cache map[int]*model.FeedVersion

	// the current batch. keys will continue to be collected until timeout is hit,
	// then everything will be sent to the fetch method and out to the listeners
	batch *feedVersionLoaderBatch

	// mutex to prevent races
	mu sync.Mutex
}

type feedVersionLoaderBatch struct {
	keys    []int
	data    []*model.FeedVersion
	error   []error
	closing bool
	done    chan struct{}
}

// Load a FeedVersion by key, batching and caching will be applied automatically
func (l *FeedVersionLoader) Load(key int) (*model.FeedVersion, error) {
	return l.LoadThunk(key)()
}

// LoadThunk returns a function that when called will block waiting for a FeedVersion.
// This method should be used if you want one goroutine to make requests to many
// different data loaders without blocking until the thunk is called.
func (l *FeedVersionLoader) LoadThunk(key int) func() (*model.FeedVersion, error) {
	l.mu.Lock()
	if it, ok := l.cache[key]; ok {
		l.mu.Unlock()
		return func() (*model.FeedVersion, error) {
			return it, nil
		}
	}
	if l.batch == nil {
		l.batch = &feedVersionLoaderBatch{done: make(chan struct{})}
	}
	batch := l.batch
	pos := batch.keyIndex(l, key)
	l.mu.Unlock()

	return func() (*model.FeedVersion, error) {
		<-batch.done

		var data *model.FeedVersion
		if pos < len(batch.data) {
			data = batch.data[pos]
		}

		var err error
		// its convenient to be able to return a single error for everything
		if len(batch.error) == 1 {
			err = batch.error[0]
		} else if batch.error != nil {
			err = batch.error[pos]
		}

		if err == nil {
			l.mu.Lock()
			l.unsafeSet(key, data)
			l.mu.Unlock()
		}

		return data, err
	}
}

// LoadAll fetches many keys at once. It will be broken into appropriate sized
// sub batches depending on how the loader is configured
func (l *FeedVersionLoader) LoadAll(keys []int) ([]*model.FeedVersion, []error) {
	results := make([]func() (*model.FeedVersion, error), len(keys))

	for i, key := range keys {
		results[i] = l.LoadThunk(key)
	}

	feedVersions := make([]*model.FeedVersion, len(keys))
	errors := make([]error, len(keys))
	for i, thunk := range results {
		feedVersions[i], errors[i] = thunk()
	}
	return feedVersions, errors
}

// LoadAllThunk returns a function that when called will block waiting for a FeedVersions.
// This method should be used if you want one goroutine to make requests to many
// different data loaders without blocking until the thunk is called.
func (l *FeedVersionLoader) LoadAllThunk(keys []int) func() ([]*model.FeedVersion, []error) {
	results := make([]func() (*model.FeedVersion, error), len(keys))
	for i, key := range keys {
		results[i] = l.LoadThunk(key)
	}
	return func() ([]*model.FeedVersion, []error) {
		feedVersions := make([]*model.FeedVersion, len(keys))
		errors := make([]error, len(keys))
		for i, thunk := range results {
			feedVersions[i], errors[i] = thunk()
		}
		return feedVersions, errors
	}
}

// Prime the cache with the provided key and value. If the key already exists, no change is made
// and false is returned.
// (To forcefully prime the cache, clear the key first with loader.clear(key).prime(key, value).)
func (l *FeedVersionLoader) Prime(key int, value *model.FeedVersion) bool {
	l.mu.Lock()
	var found bool
	if _, found = l.cache[key]; !found {
		// make a copy when writing to the cache, its easy to pass a pointer in from a loop var
		// and end up with the whole cache pointing to the same value.
		cpy := *value
		l.unsafeSet(key, &cpy)
	}
	l.mu.Unlock()
	return !found
}

// Clear the value at key from the cache, if it exists
func (l *FeedVersionLoader) Clear(key int) {
	l.mu.Lock()
	delete(l.cache, key)
	l.mu.Unlock()
}

func (l *FeedVersionLoader) unsafeSet(key int, value *model.FeedVersion) {
	if l.cache == nil {
		l.cache = map[int]*model.FeedVersion{}
	}
	l.cache[key] = value
}

// keyIndex will return the location of the key in the batch, if its not found
// it will add the key to the batch
func (b *feedVersionLoaderBatch) keyIndex(l *FeedVersionLoader, key int) int {
	for i, existingKey := range b.keys {
		if key == existingKey {
			return i
		}
	}

	pos := len(b.keys)
	b.keys = append(b.keys, key)
	if pos == 0 {
		go b.startTimer(l)
	}

	if l.maxBatch != 0 && pos >= l.maxBatch-1 {
		if !b.closing {
			b.closing = true
			l.batch = nil
			go b.end(l)
		}
	}

	return pos
}

func (b *feedVersionLoaderBatch) startTimer(l *FeedVersionLoader) {
	time.Sleep(l.wait)
	l.mu.Lock()

	// we must have hit a batch limit and are already finalizing this batch
	if b.closing {
		l.mu.Unlock()
		return
	}

	l.batch = nil
	l.mu.Unlock()

	b.end(l)
}

func (b *feedVersionLoaderBatch) end(l *FeedVersionLoader) {
	b.data, b.error = l.fetch(b.keys)
	close(b.done)
}