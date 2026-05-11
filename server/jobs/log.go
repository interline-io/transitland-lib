package jobs

import (
	"context"
	"time"

	"github.com/interline-io/log"
	"github.com/rs/zerolog"
)

// NewRunLogger logs job start/error/completion under a per-job logger keyed
// by Kind/Args. Register on a Runner via Use.
func NewRunLogger(logger zerolog.Logger) Middleware {
	return func(w Worker, j Job) Worker {
		return &runLogger{log: logger, job: j, Worker: w}
	}
}

type runLogger struct {
	log zerolog.Logger
	job Job
	Worker
}

func (w *runLogger) Run(ctx context.Context) error {
	ctxLog := log.For(ctx).With().Str("kind", w.job.Kind).Any("args", w.job.Args).Logger()
	ctx = ctxLog.WithContext(ctx)
	t1 := time.Now()
	ctxLog.Info().Msg("job: started")
	if err := w.Worker.Run(ctx); err != nil {
		ctxLog.Error().Err(err).Msg("job: error")
		return err
	}
	ctxLog.Info().Int64("job_time_ms", time.Since(t1).Milliseconds()).Msg("job: completed")
	return nil
}
