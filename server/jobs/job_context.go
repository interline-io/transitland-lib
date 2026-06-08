package jobs

import "context"

// jobIDCtxKey carries the currently-executing job's ID in a Worker's context.
// The Runner installs it (see Runner.Run) so workers and per-job helpers (e.g.
// the artifact store) can address the job that is producing their output without
// the ID being threaded through Args.
type jobIDCtxKey struct{}

// WithJobID returns a context carrying the executing job's ID.
func WithJobID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, jobIDCtxKey{}, id)
}

// JobIDFromContext returns the executing job's ID, or "" if none is set.
//
// The ID is populated by Runner.Run for in-process dispatch (local, River, and
// the synchronous /run path). For the Argo backend the workflow name is threaded
// into the pod and assigned to Job.ID before Runner.Run, so it is available here
// too. Backends that never assign an ID (e.g. fire-and-forget Redis) leave this
// empty, and artifact creation will fail loudly rather than write an
// unaddressable row.
func JobIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(jobIDCtxKey{}).(string); ok {
		return v
	}
	return ""
}
