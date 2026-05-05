package model

import (
	"context"
	"net/http"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/clock"
	"github.com/interline-io/transitland-lib/server/jobs"
)

type Config struct {
	Finder                   Finder
	RTFinder                 RTFinder
	GbfsFinder               GbfsFinder
	Checker                  Checker
	Actions                  Actions
	JobQueue                 jobs.JobQueue
	Clock                    clock.Clock
	Secrets                  []dmfr.Secret
	ValidateLargeFiles       bool
	DisableImage             bool
	UseMaterialized          bool
	AllowHTTPFetchUnfiltered bool
	// IncludePublic, when true, includes feed_states.public = true rows in
	// read queries regardless of the caller's per-feed permissions. This is
	// the deployment-wide policy for public-feed visibility. When false,
	// callers see only feeds explicitly granted to them by the Checker
	// (or all rows if the Checker reports global admin).
	IncludePublic           bool
	RestPrefix              string
	Storage                 string
	RTStorage               string
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

func WithConfig(ctx context.Context, cfg Config) context.Context {
	r := context.WithValue(ctx, finderCtxKey, cfg)
	return r
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
	return AddPerms(cfg.Checker, cfg.IncludePublic)(AddConfig(cfg)(next))
}
