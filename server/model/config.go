package model

import (
	"context"
	"net/http"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/feedmanager"
	"github.com/interline-io/transitland-lib/internal/clock"
	"github.com/interline-io/transitland-lib/server/jobs"
	"github.com/interline-io/transitland-lib/tldb"
)

type Config struct {
	Finder     Finder
	RTFinder   RTFinder
	GbfsFinder GbfsFinder
	Checker    Checker
	Actions    Actions
	// Adapter is the DB handle for the operations that are inherently
	// database-backed (editor entity CRUD, feed-version DB readers, fleet
	// maintenance) and the FeedManager for the import/fetch bookkeeping flows.
	// Both are set only in DB-backed deployments; an in-memory finder leaves
	// them nil and those operations are simply unavailable. This replaces the
	// former Finder.DBX() escape hatch, which forced every Finder (including the
	// in-memory one) to expose a DB handle it might not have.
	Adapter     tldb.Adapter
	FeedManager feedmanager.FeedManager
	Jobs        jobs.Backend
	JobRunner   *jobs.Runner
	// JobPolicy gates the synchronous /run endpoint (which doesn't go
	// through a Queue). Nil means no kind-level RBAC on /run.
	JobPolicy                jobs.AccessPolicy
	Clock                    clock.Clock
	Secrets                  []dmfr.Secret
	ValidateLargeFiles       bool
	UseMaterialized          bool
	UseGeohashFilter         bool
	AllowHTTPFetchUnfiltered bool
	RestPrefix               string
	// JobsPrefix is the public prefix of the jobserver mount (analogue of
	// RestPrefix for the REST mount), used to build absolute artifact download
	// links that are correct behind a path-rewriting ingress. Empty yields
	// host-relative links.
	JobsPrefix      string
	Storage         string
	RTStorage       string
	ArtifactStorage string // job-artifact storage URL; no fallback to Storage
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
	// Treat an empty storage URL as "not configured", matching the jobserver's
	// requireArtifactReader: NewStore returns a non-nil *Store even for an empty
	// URL (writes then fail loudly), so guarding only the factory would hand a
	// worker a handle that errors on every write.
	if cfg.ArtifactStoreFactory == nil || cfg.ArtifactStorage == "" {
		return nil
	}
	m, ok := jobs.JobMetaFromContext(ctx)
	if !ok {
		return nil
	}
	return cfg.ArtifactStoreFactory.For(m.ID, m.UserID, m.Kind)
}

// NewConfigMiddleware installs cfg into the job context, the jobs-side analogue
// of AddConfig. Register it on a Runner (runner.Use) so background workers can
// resolve the Config via ForContext. Always overwrites any inbound cfg.
func NewConfigMiddleware(cfg Config) jobs.Middleware {
	return func(w jobs.Worker, _ jobs.Job) jobs.Worker {
		return &configWorker{cfg: cfg, Worker: w}
	}
}

type configWorker struct {
	jobs.Worker
	cfg Config
}

func (w *configWorker) Run(ctx context.Context) error {
	return w.Worker.Run(WithConfig(ctx, w.cfg))
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
