package model

import (
	"context"
	"net/http"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/clock"
	"github.com/interline-io/transitland-lib/server/jobs"
)

type Config struct {
	Finder                  Finder
	RTFinder                RTFinder
	GbfsFinder              GbfsFinder
	Checker                 Checker
	Actions                 Actions
	JobQueue                jobs.JobQueue
	Clock                   clock.Clock
	Secrets                 []dmfr.Secret
	ValidateLargeFiles      bool
	DisableImage            bool
	UseMaterialized         bool
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
