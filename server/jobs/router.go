package jobs

import (
	"context"
	"sync"
)

// Router is a Backend that aggregates other Backends. Queue(name) routes to
// whichever Backend hosts the named queue; Run/Stop fan out across the
// unique underlying Backends.
//
// The whole River-vs-Argo-vs-Local split is one declaration in one place:
//
//	router := jobs.NewRouter(map[string]jobs.Backend{
//	    "rt-fetch":            riverBackend,
//	    "static-fetch":        riverBackend,
//	    "feed-version-import": argoBackend,
//	})
type Router struct {
	routes map[string]Backend // queue name → owning backend
	unique []Backend          // dedup'd lifecycle list
}

// NewRouter constructs a Router from a queue→Backend map. Backends that
// appear under multiple queue names are deduped for lifecycle calls.
func NewRouter(routes map[string]Backend) *Router {
	r := &Router{
		routes: make(map[string]Backend, len(routes)),
	}
	seen := map[Backend]bool{}
	for name, b := range routes {
		if b == nil {
			continue
		}
		r.routes[name] = b
		if !seen[b] {
			seen[b] = true
			r.unique = append(r.unique, b)
		}
	}
	return r
}

// Queue forwards to the Backend registered for the named queue, or returns
// nil if no Backend hosts it.
func (r *Router) Queue(name string) Queue {
	b, ok := r.routes[name]
	if !ok {
		return nil
	}
	return b.Queue(name)
}

// Run starts every unique underlying Backend and blocks until all return.
// Each Backend's Run typically blocks until its Stop is called.
func (r *Router) Run(ctx context.Context) error {
	if len(r.unique) == 0 {
		<-ctx.Done()
		return nil
	}
	var wg sync.WaitGroup
	errs := make(chan error, len(r.unique))
	for _, b := range r.unique {
		b := b
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := b.Run(ctx); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// Stop signals every underlying Backend to shut down. Returns the first
// error encountered (subsequent errors are dropped).
func (r *Router) Stop(ctx context.Context) error {
	var firstErr error
	for _, b := range r.unique {
		if err := b.Stop(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
