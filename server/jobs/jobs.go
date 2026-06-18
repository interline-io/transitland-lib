package jobs

import (
	"context"
	"errors"
	"time"

	"github.com/interline-io/transitland-lib/server/auth/authn"
)

// Args is the JSON-marshalable payload a Worker is constructed from.
type Args map[string]any

// Job is a single unit of work. Kind matches Worker.Kind() for routing; Args
// is the JSON payload deserialized into the worker; Opts holds infrastructure
// fields (auth, scheduling, dedup). ID is queue-assigned on Submit; any
// caller-set value is overwritten.
type Job struct {
	ID   string  `json:"id"`
	Kind string  `json:"kind" river:"unique"`
	Args Args    `json:"args" river:"unique"`
	Opts JobOpts `json:"opts"`
}

// JobOpts carries infrastructure-level submit options.
//
// Deadline: jobs picked up after the deadline are cancelled instead of run;
// zero means no deadline.
// Unique: deduplicates by (Kind, Args) while a matching job is queued or
// running — no concurrent runs.
type JobOpts struct {
	UserID   string    `json:"user_id"`
	Deadline time.Time `json:"deadline"`
	Unique   bool      `json:"unique"`
}

type JobState string

const (
	JobStateQueued    JobState = "queued"
	JobStateRunning   JobState = "running"
	JobStateSucceeded JobState = "succeeded"
	JobStateFailed    JobState = "failed"
	JobStateCancelled JobState = "cancelled"
)

func (s JobState) Terminal() bool {
	return s == JobStateSucceeded || s == JobStateFailed || s == JobStateCancelled
}

// Sentinel errors returned by Backend / Queue methods.
var (
	ErrJobAccessDenied = errors.New("job access denied")
	ErrJobNotFound     = errors.New("job not found")
	ErrUnknownQueue    = errors.New("unknown queue")
)

type JobStatus struct {
	State       JobState   `json:"state"`
	Job         Job        `json:"job"`
	SubmittedAt time.Time  `json:"submitted_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	Error       string     `json:"error,omitempty"`
	Attempt     int        `json:"attempt,omitempty"`
}

type JobEvent struct {
	JobID   string    `json:"job_id"`
	State   JobState  `json:"state"`
	Attempt int       `json:"attempt,omitempty"`
	Message string    `json:"message,omitempty"`
	Time    time.Time `json:"time"`
}

// ListOptions controls List filtering and paging. Offset is a zero-based
// row offset into the sorted result; pair with Limit for paging.
type ListOptions struct {
	States []JobState
	UserID string
	Kind   string
	Limit  int
	Offset int
}

type ListResult struct {
	Jobs []JobStatus `json:"jobs"`
}

// AccessPolicy gates Submit, per-job reads (Status/Watch/Cancel), and List
// scoping. Backends call CanSubmit at the top of Submit, CanRead before
// returning per-job state, and ScopeList before executing a List. Pass a
// custom implementation to the backend constructor for per-Kind RBAC or
// stricter rules.
type AccessPolicy interface {
	CanSubmit(ctx context.Context, job Job) error
	CanRead(ctx context.Context, status JobStatus) error
	ScopeList(ctx context.Context, opts ListOptions) (ListOptions, error)
}

// CreatorOrAdmin is the default AccessPolicy: admins see everything; everyone
// else sees only jobs they created (matched by Opts.UserID). Any authenticated
// caller may submit a job of any Kind.
type CreatorOrAdmin struct{}

func (CreatorOrAdmin) CanSubmit(ctx context.Context, _ Job) error {
	if authn.ForContext(ctx) == nil {
		return ErrJobAccessDenied
	}
	return nil
}

func (CreatorOrAdmin) CanRead(ctx context.Context, status JobStatus) error {
	user := authn.ForContext(ctx)
	if user == nil {
		return ErrJobAccessDenied
	}
	if user.HasRole("admin") {
		return nil
	}
	if user.ID() != "" && user.ID() == status.Job.Opts.UserID {
		return nil
	}
	return ErrJobAccessDenied
}

// ScopeList force-overrides opts.UserID to the caller's ID for non-admins,
// so they cannot query other users' jobs.
func (CreatorOrAdmin) ScopeList(ctx context.Context, opts ListOptions) (ListOptions, error) {
	user := authn.ForContext(ctx)
	if user == nil {
		return opts, ErrJobAccessDenied
	}
	if user.HasRole("admin") {
		return opts, nil
	}
	if user.ID() == "" {
		return opts, ErrJobAccessDenied
	}
	opts.UserID = user.ID()
	return opts, nil
}

// Worker defines a job worker. The same Worker code runs whether the work
// is dispatched in-process via a Backend's worker pool or scheduled in a
// remote pod (e.g. an Argo workflow whose container calls Runner.Run).
type Worker interface {
	Kind() string
	Run(context.Context) error
}

type WorkerFn func() Worker

// Middleware wraps a Worker for one execution. Registered on Runner via Use.
// Executes in registration order: the first Use() runs first on entry,
// matching the chi/net-http middleware convention.
type Middleware func(Worker, Job) Worker

// Queue is the per-queue handle returned by Backend.Queue(name). Submit
// targets exactly the named queue — no Job.Queue field exists. Backends may
// return a Queue that also implements StatusQueue and/or PeriodicQueue.
type Queue interface {
	Submit(context.Context, Job) (JobStatus, error)
	SubmitMany(context.Context, []Job) ([]JobStatus, error)
}

// StatusQueue is the optional capability for queues that track individual
// jobs after submission. Fire-and-forget queues (e.g. Redis) return a Queue
// that doesn't implement it; callers type-assert before use.
//
// Watch emits at least one terminal JobEvent before closing. Adapters MAY
// emit intermediate transitions but aren't required to (River only emits
// terminal). Drain until close and trust the last event for final state.
//
// Cancel is idempotent on terminal jobs. Backends that can't interrupt
// running work return without affecting the in-progress run; queued
// cancellation must always succeed.
type StatusQueue interface {
	Queue
	Status(context.Context, string) (JobStatus, error)
	Watch(context.Context, string) (<-chan JobEvent, error)
	List(context.Context, ListOptions) (ListResult, error)
	Cancel(context.Context, string) error
}

// PeriodicQueue is the optional capability for recurring jobs. cronTab
// (non-empty) takes precedence over period; otherwise period is a fixed
// interval. AddPeriodic returns an opaque ID for RemovePeriodic.
type PeriodicQueue interface {
	Queue
	AddPeriodic(ctx context.Context, jobFunc func() Job, period time.Duration, cronTab string) (string, error)
	RemovePeriodic(ctx context.Context, periodicJobId string) error
}

// Backend hosts one or more named queues. Get a per-queue handle via
// Queue(name) — returns ErrUnknownQueue if the backend doesn't host that
// queue.
//
// Lifecycle:
//   - Run blocks the calling goroutine until shutdown completes.
//   - Shutdown is the graceful primitive: triggers drain (stop accepting new
//     jobs, let in-flight run to natural completion) and blocks until the
//     drain finishes or ctx fires. Mirrors net/http.Server.Shutdown.
//   - Stop is the hard cancel: signals shutdown by cancelling the backend
//     context, which propagates to in-flight worker contexts. Returns
//     immediately; pair with Wait if you need to observe completion.
//   - Wait blocks until Run has returned (workers exited and shutdown
//     finished), or until ctx fires. Wait does NOT trigger shutdown.
//
// Typical pattern (mirrors net/http):
//
//	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	if err := backend.Shutdown(shutdownCtx); err != nil {
//	    backend.Stop(context.Background()) // hard fallback on timeout
//	}
//
// Implementations: LocalBackend (in-process), RiverBackend (Postgres),
// ArgoBackend (k8s workflows), RedisBackend (fire-and-forget — Shutdown is
// a no-op since there's nothing to drain on the producer side). A Router is
// itself a Backend that aggregates other Backends and dispatches Queue(name)
// to whichever Backend hosts the named queue.
type Backend interface {
	Queue(name string) (Queue, error)
	Run(context.Context) error
	Shutdown(ctx context.Context) error
	Stop(context.Context) error
	Wait(ctx context.Context) error
}
