package model

import (
	"context"
	"net/http"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/clock"
	"github.com/interline-io/transitland-lib/server/jobs"
)

type Config struct {
	Finder     Finder
	RTFinder   RTFinder
	GbfsFinder GbfsFinder
	Checker    Checker
	Actions    Actions
	Jobs       jobs.Backend
	JobRunner  *jobs.Runner
	// JobPolicy gates the synchronous /run endpoint (which doesn't go
	// through a Queue). Nil means no kind-level RBAC on /run.
	JobPolicy                jobs.AccessPolicy
	Clock                    clock.Clock
	Secrets                  []dmfr.Secret
	ValidateLargeFiles       bool
	DisableImage             bool
	UseMaterialized          bool
	UseGeohashFilter         bool
	AllowHTTPFetchUnfiltered bool
	RestPrefix               string
	Storage                  string
	RTStorage                string
	ArtifactStorage          string // job-artifact storage URL; no fallback to Storage
	// ArtifactStoreFactory is the unscoped read/serve side (jobserver) and the
	// producer of per-job scoped handles (see JobArtifacts). The per-job handle
	// is intentionally NOT a Config field: it is execution-scoped, resolved from
	// the job's JobMeta rather than stored on this process-wide struct.
	ArtifactStoreFactory    ArtifactStoreFactory
	LoaderBatchSize         int
	LoaderStopTimeBatchSize int
	MaxRadius               float64
}

var finderCtxKey = &contextKey{"finderConfig"}

type contextKey struct {
	name string
}

func ForContext(ctx context.Context) Config {
	raw, ok := ctx.Value(finderCtxKey).(Config)
	if !ok {
		return Config{}
	}
	return raw
}

// WithConfig stores cfg in ctx for non-HTTP entry points (background jobs,
// tests). Unlike AddConfig, this does not enforce a non-nil Checker — test
// scaffolding (testconfig) is responsible for providing one.
func WithConfig(ctx context.Context, cfg Config) context.Context {
	r := context.WithValue(ctx, finderCtxKey, cfg)
	return r
}

// JobArtifacts returns an ArtifactStore scoped to the executing job, or nil when
// there is no job on the context (e.g. an HTTP request) or no artifact storage
// is configured for this deployment. Workers call this to publish files
// attributed to the job they are running; the scope (id/user/kind) comes from
// the runner-stamped JobMeta, so a worker cannot misattribute a file to another
// job. A non-nil return means "in a job AND artifacts are available here."
func JobArtifacts(ctx context.Context) ArtifactStore {
	cfg := ForContext(ctx)
	if cfg.ArtifactStoreFactory == nil {
		return nil
	}
	m, ok := jobs.JobMetaFromContext(ctx)
	if !ok {
		return nil
	}
	return cfg.ArtifactStoreFactory.For(m.ID, m.UserID, m.Kind)
}

func AddConfig(cfg Config) func(http.Handler) http.Handler {
	if cfg.Checker == nil {
		panic("model.AddConfig: Config.Checker must be set; install authz.AllowAllChecker or authz.DenyAllChecker for demo/test, or a real Checker for production")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			r = r.WithContext(WithConfig(ctx, cfg))
			next.ServeHTTP(w, r)
		})
	}
}

func AddConfigAndPerms(cfg Config, next http.Handler) http.Handler {
	return AddPerms(cfg.Checker)(AddConfig(cfg)(next))
}
