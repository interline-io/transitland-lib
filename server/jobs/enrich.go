package jobs

import "context"

// Enricher returns a new context derived from the input, or an error if a
// required lookup fails. Implementations should return (ctx, nil) when
// there is nothing to enrich; only return a non-nil error when an
// enrichment lookup itself failed.
//
// The same shape is declared in server/auth/mw/enrich for the HTTP-side
// adapter; concrete enrichers satisfy both via structural typing.
type Enricher interface {
	EnrichContext(ctx context.Context) (context.Context, error)
}

// NewEnrichMiddleware adapts an Enricher to a Runner Middleware. On success
// the wrapped Worker.Run sees the enriched context. On error the wrapped
// Worker.Run is not called and the error is returned as the job result.
func NewEnrichMiddleware(e Enricher) Middleware {
	return func(next Worker, _ Job) Worker {
		return &enrichWorker{enricher: e, Worker: next}
	}
}

type enrichWorker struct {
	enricher Enricher
	Worker
}

func (w *enrichWorker) Run(ctx context.Context) error {
	ctx, err := w.enricher.EnrichContext(ctx)
	if err != nil {
		return err
	}
	return w.Worker.Run(ctx)
}
