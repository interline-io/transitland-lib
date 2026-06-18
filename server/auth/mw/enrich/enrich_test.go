package enrich

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type ctxKey string

const tagKey ctxKey = "enrich-test-tag"

// stubEnricher stamps tag onto ctx unless err is set, in which case it
// returns the error and the original context.
type stubEnricher struct {
	tag string
	err error
}

func (s stubEnricher) EnrichContext(ctx context.Context) (context.Context, error) {
	if s.err != nil {
		return ctx, s.err
	}
	return context.WithValue(ctx, tagKey, s.tag), nil
}

func TestMiddleware_Success(t *testing.T) {
	var gotTag string
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		gotTag, _ = r.Context().Value(tagKey).(string)
	})

	mw := NewMiddleware(stubEnricher{tag: "enriched"})
	rr := httptest.NewRecorder()
	mw(next).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.True(t, called, "downstream handler should be called on success")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "enriched", gotTag)
}

func TestMiddleware_ErrorReturns401(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	mw := NewMiddleware(stubEnricher{err: errors.New("lookup failed")})
	rr := httptest.NewRecorder()
	mw(next).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.False(t, called, "downstream handler must not run on enrichment error")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}
