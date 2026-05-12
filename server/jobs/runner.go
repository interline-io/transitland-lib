package jobs

import (
	"context"
	"encoding/json"
	"errors"
)

// Runner is the process-local execution engine: Worker registry + Middleware
// stack. Backends own the wire protocol and delegate execution here. The same
// Runner code is reused by remote-dispatch backends (e.g. Argo) whose pods
// import this package and call Run.
type Runner struct {
	fns         map[string]WorkerFn
	middlewares []Middleware
}

func NewRunner() *Runner {
	return &Runner{fns: map[string]WorkerFn{}}
}

func (r *Runner) Register(fn WorkerFn) error {
	w := fn()
	if w == nil {
		return errors.New("invalid worker function")
	}
	r.fns[w.Kind()] = fn
	return nil
}

func (r *Runner) Use(mw Middleware) {
	r.middlewares = append(r.middlewares, mw)
}

// Worker constructs a fresh Worker for the given Kind+Args. Most callers
// want Run; this is exposed for backends that need to apply per-execution
// wrapping before invoking Run.
func (r *Runner) Worker(kind string, args Args) (Worker, error) {
	fn, ok := r.fns[kind]
	if !ok {
		return nil, errors.New("unknown job kind: " + kind)
	}
	w := fn()
	if len(args) == 0 {
		return w, nil
	}
	blob, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(blob, w); err != nil {
		return nil, err
	}
	return w, nil
}

func (r *Runner) Run(ctx context.Context, job Job) error {
	w, err := r.Worker(job.Kind, job.Args)
	if err != nil {
		return err
	}
	if w == nil {
		return errors.New("no worker")
	}
	// Wrap in reverse so registration order matches execution order:
	// Use(A); Use(B); Use(C) → A wraps B wraps C wraps worker → on entry
	// A.Run runs first, matching the chi/net-http convention.
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		w = r.middlewares[i](w, job)
		if w == nil {
			return errors.New("middleware dropped worker")
		}
	}
	return w.Run(ctx)
}
