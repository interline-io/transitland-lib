package jobs

import (
	"context"
	"encoding/json"
	"errors"
)

// Runner is the process-local execution engine: a registry of JobWorker
// constructors plus a stack of middleware that wraps each run. Backends own
// the queue/wire protocol; they delegate the actual job execution to a Runner
// (one Runner per process, typically shared by every Backend that runs jobs
// in-process). Submit-only backends (e.g. an Argo adapter that ships work to
// pods) don't need a Runner at all.
type Runner struct {
	jobFns      map[string]JobFn
	middlewares []JobMiddleware
}

func NewRunner() *Runner {
	return &Runner{jobFns: map[string]JobFn{}}
}

func (r *Runner) RegisterWorker(jobFn JobFn) error {
	jw := jobFn()
	if jw == nil {
		return errors.New("invalid job function")
	}
	r.jobFns[jw.Kind()] = jobFn
	return nil
}

func (r *Runner) Use(mw JobMiddleware) {
	r.middlewares = append(r.middlewares, mw)
}

// Worker constructs a JobWorker for the given Kind+Args. Backends that need
// to apply additional per-execution wrapping (e.g. wire-format-specific
// tracing) can call this directly; most callers should use Run.
func (r *Runner) Worker(kind string, args JobArgs) (JobWorker, error) {
	jobFn, ok := r.jobFns[kind]
	if !ok {
		return nil, errors.New("unknown job kind")
	}
	w := jobFn()
	jw, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(jw, w); err != nil {
		return nil, err
	}
	return w, nil
}

// Run resolves the job's worker, applies registered middlewares in order, and
// invokes Run. Backends call this from their worker pool / dispatch loop.
func (r *Runner) Run(ctx context.Context, job Job) error {
	w, err := r.Worker(job.Kind, job.Args)
	if err != nil {
		return err
	}
	if w == nil {
		return errors.New("no job")
	}
	for _, mwf := range r.middlewares {
		w = mwf(w, job)
		if w == nil {
			return errors.New("no job")
		}
	}
	return w.Run(ctx)
}
