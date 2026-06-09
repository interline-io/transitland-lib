package jobs

import (
	"context"
	"time"
)

// JobMeta is the identity of the job currently executing. Runner.Run stamps it
// onto the context once per run, from the Job. It carries only primitives so it
// can live in package jobs, which must not import server/model; model-layer
// capabilities scoped to the job (e.g. the artifact store) are resolved from it
// at the model layer — see model.JobArtifacts.
type JobMeta struct {
	ID       string
	Kind     string
	UserID   string
	Deadline time.Time
}

type jobMetaKey struct{}

// WithJobMeta returns ctx carrying m. Runner.Run calls this once per execution,
// so every worker (in-process or remote-pod) sees its own identity uniformly.
func WithJobMeta(ctx context.Context, m JobMeta) context.Context {
	return context.WithValue(ctx, jobMetaKey{}, m)
}

// JobMetaFromContext returns the executing job's identity, or ok=false when ctx
// is not inside a job run (e.g. an HTTP request).
func JobMetaFromContext(ctx context.Context) (JobMeta, bool) {
	m, ok := ctx.Value(jobMetaKey{}).(JobMeta)
	return m, ok
}
