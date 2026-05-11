package jobs

import (
	"context"
	"errors"
	"sync"
)

// Router is a Backend that dispatches per-queue traffic to one of several
// underlying Backends — e.g. River for fetch queues and Argo for imports —
// declared in a single map[queueName]Backend.
type Router struct {
	routes map[string]Backend // queue name → owning backend
	unique []Backend          // dedup'd lifecycle list
}

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

func (r *Router) Queue(name string) (Queue, error) {
	b, ok := r.routes[name]
	if !ok {
		return nil, ErrUnknownQueue
	}
	return b.Queue(name)
}

func (r *Router) Run(ctx context.Context) error {
	if len(r.unique) == 0 {
		<-ctx.Done()
		return nil
	}
	errs := make([]error, len(r.unique))
	var wg sync.WaitGroup
	for i, b := range r.unique {
		i, b := i, b
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = b.Run(ctx)
		}()
	}
	wg.Wait()
	return errors.Join(errs...)
}

func (r *Router) Stop(ctx context.Context) error {
	errs := make([]error, len(r.unique))
	for i, b := range r.unique {
		errs[i] = b.Stop(ctx)
	}
	return errors.Join(errs...)
}

// Wait fans out to every underlying Backend's Wait. Returns the joined errors;
// individual backends that exceed ctx return ctx.Err() but the others may still
// drain cleanly.
func (r *Router) Wait(ctx context.Context) error {
	errs := make([]error, len(r.unique))
	var wg sync.WaitGroup
	for i, b := range r.unique {
		i, b := i, b
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = b.Wait(ctx)
		}()
	}
	wg.Wait()
	return errors.Join(errs...)
}
