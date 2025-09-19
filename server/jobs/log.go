package jobs

import (
	"context"
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

func (w *JobLogger) AddJobType(jobFn JobFn) error {
	w.log.Trace().Str("job_type", jobFn().Kind()).Msg("jobs: adding job type")
	return w.JobQueue.AddJobType(jobFn)
}

func (w *JobLogger) AddJob(ctx context.Context, job Job) error {
	w.log.Trace().Str("job_type", job.JobType).Any("job_args", job.JobArgs).Msg("jobs: adding job")
	return w.JobQueue.AddJob(ctx, job)
}

func (w *JobLogger) AddJobs(ctx context.Context, jobs []Job) error {
	w.log.Trace().Msg("jobs: adding jobs")
	return w.JobQueue.AddJobs(ctx, jobs)
}

func (w *JobLogger) RunJob(ctx context.Context, job Job) error {
	w.log.Trace().Str("job_type", job.JobType).Any("job_args", job.JobArgs).Msg("jobs: run job")
	return w.JobQueue.RunJob(ctx, job)
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
	ctxLogger := log.For(ctx).With().Str("job_type", w.job.JobType).Any("job_args", w.job.JobArgs).Logger()

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
