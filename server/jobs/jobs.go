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

// JobQueue is the unified interface every job backend must satisfy.
//
// Watch is best-effort and intended for UI feedback. Adapters may skip
// intermediate transitions and the terminal event payload may be dropped
// under load; the channel is still guaranteed to close when the job
// reaches a terminal state. Only Status returns authoritative data —
// callers should call it after the channel closes to learn the outcome.
type JobQueue interface {
	Use(JobMiddleware)
	AddQueue(string, int) error
	AddJobType(JobFn) error
	AddJob(context.Context, Job) (JobStatus, error)
	AddJobs(context.Context, []Job) ([]JobStatus, error)
	AddPeriodicJob(context.Context, func() Job, time.Duration, string) error
	RunJob(context.Context, Job) (JobStatus, error)
	Status(context.Context, string) (JobStatus, error)
	Watch(context.Context, string) (<-chan JobEvent, error)
	ListJobs(context.Context, JobListOptions) (JobListResult, error)
	Run(context.Context) error
	Stop(context.Context) error
}

// Job defines a single job. JobId is always assigned by the adapter; any
// caller-set value is overwritten by AddJob, AddJobs, and RunJob.
type Job struct {
	JobId       string  `json:"job_id"`
	UserId      string  `json:"user_id"`
	Queue       string  `json:"queue"`
	JobType     string  `json:"job_type" river:"unique"`
	JobArgs     JobArgs `json:"job_args" river:"unique"`
	JobDeadline int64   `json:"job_deadline"`
	Unique      bool    `json:"unique"`
}

func (job *Job) HexKey() (string, error) {
	bytes, err := json.Marshal(job.JobArgs)
	if err != nil {
		return "", err
	}
	sum := sha1.Sum(bytes)
	return job.JobType + ":" + hex.EncodeToString(sum[:]), nil
}

// JobStatus is the lifecycle state of a submitted job.
type JobStatus struct {
	JobId       string     `json:"job_id"`
	UserId      string     `json:"user_id"`
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
	JobId   string    `json:"job_id"`
	State   JobState  `json:"state"`
	Message string    `json:"message,omitempty"`
	Time    time.Time `json:"time"`
}

// JobListOptions controls ListJobs filtering and paging. After is an opaque
// cursor returned in JobListResult.NextCursor; callers should pass it back
// verbatim. The encoding is adapter-specific.
type JobListOptions struct {
	States  []JobState
	UserId  string
	JobType string
	Limit   int
	After   string
}

// JobListResult is a page of JobStatus rows plus an opaque cursor for the next page.
type JobListResult struct {
	Jobs       []JobStatus
	NextCursor string
}

// CheckJobAccess returns nil if the context user is admin or matches status.UserId.
// Otherwise it returns ErrJobAccessDenied.
func CheckJobAccess(ctx context.Context, status JobStatus) error {
	user := authn.ForContext(ctx)
	if user == nil {
		return ErrJobAccessDenied
	}
	if user.HasRole("admin") {
		return nil
	}
	if user.ID() != "" && user.ID() == status.UserId {
		return nil
	}
	return ErrJobAccessDenied
}

// JobWorker defines a job worker
type JobWorker interface {
	Kind() string
	Run(context.Context) error
}

type JobFn func() JobWorker

type JobMiddleware func(JobWorker, Job) JobWorker
