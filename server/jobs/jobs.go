package jobs

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/interline-io/transitland-lib/server/auth/authn"
)

// Args is the JSON-marshalable payload a Worker is constructed from.
type Args map[string]any

// Job is a single unit of work. The owning queue is implicit — you got here
// by calling Backend.Queue(name).Submit(...). ID is always assigned by the
// queue; any caller-set value is overwritten. Kind matches Worker.Kind() —
// that's how Runner routes a job to its worker.
//
// Unique deduplicates submissions with the same Kind+Args within the queue.
// UniqueWindow controls the dedup span: zero means "while a matching job is
// still queued or running" (no concurrent runs); a positive duration means
// "at most one matching job submitted in the trailing window" (cron-style).
type Job struct {
	ID           string        `json:"id"`
	UserID       string        `json:"user_id"`
	Kind         string        `json:"kind" river:"unique"`
	Args         Args          `json:"args" river:"unique"`
	Deadline     int64         `json:"deadline"`
	Unique       bool          `json:"unique"`
	UniqueWindow time.Duration `json:"unique_window,omitempty"`
}

// HexKey is a stable hash of (Kind, Args) used by backends to dedup unique
// jobs within a queue. Queue is not part of the key — uniqueness is scoped
// per-queue by the backend's storage.
func (job *Job) HexKey() (string, error) {
	bytes, err := json.Marshal(job.Args)
	if err != nil {
		return "", err
	}
	sum := sha1.Sum(bytes)
	return job.Kind + ":" + hex.EncodeToString(sum[:]), nil
}

// JobState describes a job's lifecycle state.
type JobState string

const (
	JobStateQueued    JobState = "queued"
	JobStateRunning   JobState = "running"
	JobStateSucceeded JobState = "succeeded"
	JobStateFailed    JobState = "failed"
	JobStateCancelled JobState = "cancelled"
	JobStateUnknown   JobState = "unknown"
)

// Terminal reports whether the state is final and will not transition again.
func (s JobState) Terminal() bool {
	return s == JobStateSucceeded || s == JobStateFailed || s == JobStateCancelled
}

// Sentinel errors returned by Backend / Queue methods.
var (
	ErrJobAccessDenied    = errors.New("job access denied")
	ErrJobNotFound        = errors.New("job not found")
	ErrCancelNotSupported = errors.New("job cancel not supported")
	ErrUnknownQueue       = errors.New("unknown queue")
)

// JobStatus is the lifecycle state of a submitted job.
type JobStatus struct {
	State       JobState   `json:"state"`
	Job         Job        `json:"job"`
	SubmittedAt time.Time  `json:"submitted_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	Error       string     `json:"error,omitempty"`
	Attempt     int        `json:"attempt,omitempty"`
}

// JobEvent is emitted on a state transition or status update.
type JobEvent struct {
	JobID   string    `json:"job_id"`
	State   JobState  `json:"state"`
	Attempt int       `json:"attempt,omitempty"`
	Message string    `json:"message,omitempty"`
	Time    time.Time `json:"time"`
}

// ListOptions controls List filtering and paging. After is an opaque cursor
// returned in ListResult.NextCursor; pass it back verbatim for the next page.
// The encoding is backend-specific.
type ListOptions struct {
	States []JobState
	UserID string
	Kind   string
	Limit  int
	After  string
}

// ListResult is a page of JobStatus rows plus an opaque cursor for the next page.
type ListResult struct {
	Jobs       []JobStatus
	NextCursor string
}

// CheckAccess returns nil if the context user is admin or matches the job's
// creator (status.Job.UserID). Otherwise it returns ErrJobAccessDenied.
func CheckAccess(ctx context.Context, status JobStatus) error {
	user := authn.ForContext(ctx)
	if user == nil {
		return ErrJobAccessDenied
	}
	if user.HasRole("admin") {
		return nil
	}
	if user.ID() != "" && user.ID() == status.Job.UserID {
		return nil
	}
	return ErrJobAccessDenied
}

// Worker defines a job worker. The same Worker code runs whether the work
// is dispatched in-process via a Backend's worker pool or scheduled in a
// remote pod (e.g. an Argo workflow whose container calls Runner.Run).
type Worker interface {
	Kind() string
	Run(context.Context) error
}

// WorkerFn constructs a fresh Worker. Used by Runner.Register.
type WorkerFn func() Worker

// Middleware wraps a Worker for one execution. Registered on Runner via Use.
// Runs in the order they were registered (outermost first on the way in).
type Middleware func(Worker, Job) Worker

// Queue is the per-queue handle returned by Backend.Queue(name). Submit
// targets exactly the named queue — no Job.Queue field exists. Backends may
// return a Queue that also implements StatusQueue and/or PeriodicQueue.
type Queue interface {
	Submit(context.Context, Job) (JobStatus, error)
	SubmitMany(context.Context, []Job) ([]JobStatus, error)
}

// StatusQueue is the optional capability for queues that track individual
// jobs after submission — the basis of UI status displays, ops dashboards,
// and creator-only auth. Fire-and-forget queues simply return a Queue that
// doesn't implement StatusQueue; callers should type-assert before use.
//
// Watch is best-effort and intended for UI feedback. Adapters may skip
// intermediate transitions and the terminal event payload may be dropped
// under load; the channel is still guaranteed to close when the job reaches
// a terminal state. Only Status returns authoritative data — callers should
// call it after the channel closes to learn the outcome.
//
// Cancel requests cancellation of a queued or running job. Idempotent on
// terminal jobs. Backends that can't cancel running work should return
// ErrCancelNotSupported for the running case; queued cancellation must
// always succeed.
type StatusQueue interface {
	Queue
	Status(context.Context, string) (JobStatus, error)
	Watch(context.Context, string) (<-chan JobEvent, error)
	List(context.Context, ListOptions) (ListResult, error)
	Cancel(context.Context, string) error
}

// PeriodicQueue is the optional capability for queues that support recurring
// jobs. AddPeriodic returns an opaque ID that RemovePeriodic accepts. When
// cronTab is non-empty it takes precedence over period; otherwise period is
// used as a fixed interval.
type PeriodicQueue interface {
	Queue
	AddPeriodic(ctx context.Context, jobFunc func() Job, period time.Duration, cronTab string) (string, error)
	RemovePeriodic(ctx context.Context, periodicJobId string) error
}

// Backend hosts one or more named queues. Get a per-queue handle via
// Queue(name) — returns nil if the backend doesn't host that queue.
// Backends own runtime lifecycle (Run blocks until Stop).
//
// Implementations: LocalBackend (in-process), RiverBackend (Postgres),
// ArgoBackend (k8s workflows), RedisBackend (fire-and-forget). A Router is
// itself a Backend that aggregates other Backends and dispatches Queue(name)
// to whichever Backend hosts the named queue.
type Backend interface {
	Queue(name string) Queue
	Run(context.Context) error
	Stop(context.Context) error
}
