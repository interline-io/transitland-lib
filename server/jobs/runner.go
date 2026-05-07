package jobs

import (
	"context"
	"encoding/json"
	"errors"
)

// Runner is the process-local execution engine: a registry of Worker
// constructors plus a stack of Middleware that wraps each run. Backends own
// the queue/wire protocol; they delegate the actual job execution to a
// Runner. One Runner per process is typical — shared by every Backend that
// runs jobs in-process and by the synchronous /run endpoint. Submit-only
// backends (e.g. an Argo adapter that ships work to pods) don't need a
// Runner; the pods that execute the work import the same Runner code.
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

// Worker constructs a Worker for the given Kind+Args. Backends that need to
// apply additional per-execution wrapping (e.g. wire-format-specific
// instrumentation) can call this directly; most callers should use Run.
func (r *Runner) Worker(kind string, args Args) (Worker, error) {
	fn, ok := r.fns[kind]
	if !ok {
		return nil, errors.New("unknown job kind: " + kind)
	}
	w := fn()
	blob, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(blob, w); err != nil {
		return nil, err
	}
	return w, nil
}

// Run resolves the job's Worker, applies registered middlewares in order,
// and invokes Run. Backends call this from their worker pool; the /run
// endpoint calls it directly for synchronous execution.
func (r *Runner) Run(ctx context.Context, job Job) error {
	w, err := r.Worker(job.Kind, job.Args)
	if err != nil {
		return err
	}
	if w == nil {
		return errors.New("no worker")
	}
	for _, mw := range r.middlewares {
		w = mw(w, job)
		if w == nil {
			return errors.New("middleware dropped worker")
		}
	}
	return w.Run(ctx)
}
