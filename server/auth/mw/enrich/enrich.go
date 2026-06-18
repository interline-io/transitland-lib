// Package enrich provides an HTTP middleware that runs a context-enriching
// step after the bare user has been stamped in (by JWT/header/etc.) and
// before downstream handlers run. Typical enrichers inflate roles from an
// upstream service, attach private-feed access lists, or stash a
// permissions/rate-limit struct on the context.
//
// The Enricher interface is declared here at the consumer; the same shape
// is re-declared in the jobs package for the worker-side adapter. Concrete
// enrichers satisfy both via structural typing.
package enrich

import (
	"context"
	"encoding/json"
	"net/http"
)

// Enricher returns a new context derived from the input, or an error if a
// required lookup fails. Implementations should return (ctx, nil) when
// there is nothing to enrich (for example, no user in context); only return
// a non-nil error when an enrichment lookup itself failed.
type Enricher interface {
	EnrichContext(ctx context.Context) (context.Context, error)
}

// NewMiddleware adapts an Enricher to a chi/net-http middleware. On success
// the downstream handler sees the enriched context. On error the middleware
// writes a JSON 401 and the downstream handler is not called.
func NewMiddleware(e Enricher) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, err := e.EnrichContext(r.Context())
			if err != nil {
				writeJsonError(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeJsonError(w http.ResponseWriter, msg string, statusCode int) {
	a := map[string]string{"error": msg}
	jj, _ := json.Marshal(&a)
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(jj)
}
