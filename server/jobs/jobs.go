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

type JobArgs map[string]any

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

// ErrJobAccessDenied is returned when the caller is neither admin nor the job's creator.
var ErrJobAccessDenied = errors.New("job access denied")

// ErrJobNotFound is returned when a JobId is not known to the queue.
var ErrJobNotFound = errors.New("job not found")

// ErrCancelNotSupported is returned by Cancel when the backend can observe
// the job but cannot stop it (e.g. running on a worker the queue can't reach).
var ErrCancelNotSupported = errors.New("job cancel not supported")

// Backend is the minimum interface every queue backend must satisfy:
// submit jobs and run a worker pool. Worker registration and middleware live
// on Runner — backends delegate execution by holding a *Runner. Lifecycle
// observation (Status, Watch, ListJobs, Cancel) and recurring scheduling are
// optional capabilities; see JobStatusReporter and PeriodicScheduler.
type Backend interface {
	AddQueue(string, int) error
	AddJob(context.Context, Job) (JobStatus, error)
	AddJobs(context.Context, []Job) ([]JobStatus, error)
	Run(context.Context) error
	Stop(context.Context) error
}

// JobStatusReporter is the optional capability for backends that track
// individual jobs after submission — the basis of UI status displays, ops
// dashboards, and creator-only auth. Adapters that fire-and-forget (e.g.
// Redis) simply omit it; callers should type-assert before use.
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
type JobStatusReporter interface {
	Status(context.Context, string) (JobStatus, error)
	Watch(context.Context, string) (<-chan JobEvent, error)
	ListJobs(context.Context, JobListOptions) (JobListResult, error)
	Cancel(context.Context, string) error
}

// Job defines a single job. ID is always assigned by the adapter; any
// caller-set value is overwritten by AddJob, AddJobs, and RunJob.
// Kind matches JobWorker.Kind() — that's how the queue routes a job to its worker.
//
// Unique deduplicates submissions with the same Kind+Args. UniqueWindow
// controls the dedup span: zero means "while a matching job is still in
// queued or running state" (no concurrent runs); a positive duration means
// "at most one matching job submitted in the trailing window" (cron-style).
type Job struct {
	ID           string        `json:"id"`
	UserID       string        `json:"user_id"`
	Queue        string        `json:"queue"`
	Kind         string        `json:"kind" river:"unique"`
	Args         JobArgs       `json:"args" river:"unique"`
	Deadline     int64         `json:"deadline"`
	Unique       bool          `json:"unique"`
	UniqueWindow time.Duration `json:"unique_window,omitempty"`
}

func (job *Job) HexKey() (string, error) {
	bytes, err := json.Marshal(job.Args)
	if err != nil {
		return "", err
	}
	sum := sha1.Sum(bytes)
	return job.Kind + ":" + hex.EncodeToString(sum[:]), nil
}

// JobStatus is the lifecycle state of a submitted job. JobId and UserId are
// available via Job.JobId and Job.UserId.
type JobStatus struct {
	State       JobState   `json:"state"`
	Job         Job        `json:"job"`
	SubmittedAt time.Time  `json:"submitted_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	Error       string     `json:"error,omitempty"`
	Attempt     int        `json:"attempt,omitempty"`
}

// JobEvent is emitted on a job state transition or status update.
type JobEvent struct {
	JobID   string    `json:"job_id"`
	State   JobState  `json:"state"`
	Attempt int       `json:"attempt,omitempty"`
	Message string    `json:"message,omitempty"`
	Time    time.Time `json:"time"`
}

// JobListOptions controls ListJobs filtering and paging. After is an opaque
// cursor returned in JobListResult.NextCursor; callers should pass it back
// verbatim. The encoding is adapter-specific.
type JobListOptions struct {
	States []JobState
	UserID string
	Kind   string
	Limit  int
	After  string
}

// JobListResult is a page of JobStatus rows plus an opaque cursor for the next page.
type JobListResult struct {
	Jobs       []JobStatus
	NextCursor string
}

// CheckJobAccess returns nil if the context user is admin or matches the job's
// creator (status.Job.UserId). Otherwise it returns ErrJobAccessDenied.
func CheckJobAccess(ctx context.Context, status JobStatus) error {
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

// PeriodicScheduler is an optional capability backends can implement to
// support recurring jobs. Adapters that don't (e.g. Redis) simply omit it;
// callers should type-assert before use.
//
// AddPeriodicJob returns an opaque ID that RemovePeriodicJob accepts. When
// cronTab is non-empty it takes precedence over period; otherwise period is
// used as a fixed interval.
type PeriodicScheduler interface {
	AddPeriodicJob(ctx context.Context, jobFunc func() Job, period time.Duration, cronTab string) (string, error)
	RemovePeriodicJob(ctx context.Context, periodicJobId string) error
}

// JobWorker defines a job worker
type JobWorker interface {
	Kind() string
	Run(context.Context) error
}

type JobFn func() JobWorker

type JobMiddleware func(JobWorker, Job) JobWorker
