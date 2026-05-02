package jobs

import (
	"context"
	"time"

	"github.com/interline-io/log"
	"github.com/rs/zerolog"
)

// NewJobRunLogger returns a JobMiddleware that logs each job's start, error,
// and completion (with elapsed time) under a per-job logger keyed by Kind/Args.
// Register it on a Runner via runner.Use.
func NewJobRunLogger(logger zerolog.Logger) JobMiddleware {
	return func(jw JobWorker, j Job) JobWorker {
		return &jobRunLogger{
			log:       logger,
			job:       j,
			JobWorker: jw,
		}
	}
}

type jobRunLogger struct {
	log zerolog.Logger
	job Job
	JobWorker
}

func (w *jobRunLogger) Run(ctx context.Context) error {
	ctxLogger := log.For(ctx).With().Str("kind", w.job.Kind).Any("args", w.job.Args).Logger()
	ctx = ctxLogger.WithContext(ctx)
	t1 := time.Now()
	ctxLogger.Info().Msg("job: started")
	if err := w.JobWorker.Run(ctx); err != nil {
		ctxLogger.Error().Err(err).Msg("job: error")
		return err
	}
	ctxLogger.Info().Int64("job_time_ms", (time.Now().UnixNano()-t1.UnixNano())/1e6).Msg("job: completed")
	return nil
}
