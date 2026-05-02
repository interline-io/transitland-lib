package jobs

import (
	"context"
	"errors"
	"time"

	"github.com/interline-io/log"
	"github.com/rs/zerolog"
)

func init() {
	var _ JobQueue = &JobLogger{}
}

type JobLogger struct {
	log zerolog.Logger
	JobQueue
}

func NewJobLogger(jq JobQueue) *JobLogger {
	logger := log.With().Logger()
	jq.Use(NewJobRunLogger(logger))
	return &JobLogger{
		log:      logger,
		JobQueue: jq,
	}
}

func (w *JobLogger) Use(jmw JobMiddleware) {
	w.log.Trace().Msg("jobs: using middleware")
	w.JobQueue.Use(jmw)
}

func (w *JobLogger) AddQueue(queue string, workers int) error {
	w.log.Trace().Str("queue", queue).Int("workers", workers).Msg("jobs: adding queue")
	return w.JobQueue.AddQueue(queue, workers)
}

func (w *JobLogger) RegisterWorker(jobFn JobFn) error {
	w.log.Trace().Str("kind", jobFn().Kind()).Msg("jobs: registering worker")
	return w.JobQueue.RegisterWorker(jobFn)
}

func (w *JobLogger) AddJob(ctx context.Context, job Job) (JobStatus, error) {
	w.log.Trace().Str("kind", job.Kind).Any("args", job.Args).Msg("jobs: adding job")
	return w.JobQueue.AddJob(ctx, job)
}

func (w *JobLogger) AddJobs(ctx context.Context, jobs []Job) ([]JobStatus, error) {
	w.log.Trace().Msg("jobs: adding jobs")
	return w.JobQueue.AddJobs(ctx, jobs)
}

func (w *JobLogger) RunJob(ctx context.Context, job Job) (JobStatus, error) {
	w.log.Trace().Str("kind", job.Kind).Any("args", job.Args).Msg("jobs: run job")
	return w.JobQueue.RunJob(ctx, job)
}

// AddPeriodicJob forwards to the inner queue if it implements PeriodicScheduler.
// JobLogger always satisfies the interface so callers can type-assert through
// the wrapper; the runtime error surfaces backends that don't support it.
func (w *JobLogger) AddPeriodicJob(ctx context.Context, jobFunc func() Job, period time.Duration, cronTab string) (string, error) {
	ps, ok := w.JobQueue.(PeriodicScheduler)
	if !ok {
		return "", errors.New("backend does not support periodic jobs")
	}
	w.log.Trace().Str("cron", cronTab).Dur("period", period).Msg("jobs: adding periodic job")
	return ps.AddPeriodicJob(ctx, jobFunc, period, cronTab)
}

func (w *JobLogger) RemovePeriodicJob(ctx context.Context, id string) error {
	ps, ok := w.JobQueue.(PeriodicScheduler)
	if !ok {
		return errors.New("backend does not support periodic jobs")
	}
	w.log.Trace().Str("periodic_job_id", id).Msg("jobs: removing periodic job")
	return ps.RemovePeriodicJob(ctx, id)
}

// Status, Watch, ListJobs, and Cancel forward to the inner queue if it
// implements JobStatusReporter. JobLogger always satisfies the interface so
// callers can type-assert through the wrapper; the runtime error surfaces
// backends that don't track jobs (e.g. fire-and-forget Redis).

func (w *JobLogger) Status(ctx context.Context, jobId string) (JobStatus, error) {
	r, ok := w.JobQueue.(JobStatusReporter)
	if !ok {
		return JobStatus{}, errors.New("backend does not support job status")
	}
	return r.Status(ctx, jobId)
}

func (w *JobLogger) Watch(ctx context.Context, jobId string) (<-chan JobEvent, error) {
	r, ok := w.JobQueue.(JobStatusReporter)
	if !ok {
		return nil, errors.New("backend does not support job status")
	}
	return r.Watch(ctx, jobId)
}

func (w *JobLogger) ListJobs(ctx context.Context, opts JobListOptions) (JobListResult, error) {
	r, ok := w.JobQueue.(JobStatusReporter)
	if !ok {
		return JobListResult{}, errors.New("backend does not support job status")
	}
	return r.ListJobs(ctx, opts)
}

func (w *JobLogger) Cancel(ctx context.Context, jobId string) error {
	r, ok := w.JobQueue.(JobStatusReporter)
	if !ok {
		return errors.New("backend does not support job status")
	}
	w.log.Trace().Str("job_id", jobId).Msg("jobs: cancel")
	return r.Cancel(ctx, jobId)
}

func (w *JobLogger) Run(ctx context.Context) error {
	w.log.Trace().Msg("jobs: run")
	return w.JobQueue.Run(ctx)
}

func (w *JobLogger) Stop(ctx context.Context) error {
	w.log.Trace().Msg("jobs: stop")
	return w.JobQueue.Stop(ctx)
}

//

type JobRunLogger struct {
	log zerolog.Logger
	job Job
	JobWorker
}

func (w *JobRunLogger) Run(ctx context.Context) error {
	// Create logger for this job
	ctxLogger := log.For(ctx).With().Str("kind", w.job.Kind).Any("args", w.job.Args).Logger()

	// Attach to the context
	ctx = ctxLogger.WithContext(ctx)

	// Run next job
	t1 := time.Now()
	ctxLogger.Info().Msg("job: started")
	if err := w.JobWorker.Run(ctx); err != nil {
		ctxLogger.Error().Err(err).Msg("job: error")
		return err
	}
	ctxLogger.Info().Int64("job_time_ms", (time.Now().UnixNano()-t1.UnixNano())/1e6).Msg("job: completed")
	return nil

}

func NewJobRunLogger(logger zerolog.Logger) JobMiddleware {
	return func(jw JobWorker, j Job) JobWorker {
		return &JobRunLogger{
			log:       logger,
			job:       j,
			JobWorker: jw,
		}
	}
}
