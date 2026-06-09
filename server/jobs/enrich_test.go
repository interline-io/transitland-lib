package jobs

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type ctxKey string

const tagKey ctxKey = "enrich-test-tag"

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

type stubWorker struct {
	called bool
	gotTag string
	retErr error
}

func (s *stubWorker) Kind() string { return "stub" }

func (s *stubWorker) Run(ctx context.Context) error {
	s.called = true
	s.gotTag, _ = ctx.Value(tagKey).(string)
	return s.retErr
}

func TestEnrichMiddleware_Success(t *testing.T) {
	worker := &stubWorker{}
	mw := NewEnrichMiddleware(stubEnricher{tag: "enriched"})
	wrapped := mw(worker, Job{})

	err := wrapped.Run(context.Background())
	assert.NoError(t, err)
	assert.True(t, worker.called, "wrapped worker should run on success")
	assert.Equal(t, "enriched", worker.gotTag)
}

func TestEnrichMiddleware_ErrorBlocksWorker(t *testing.T) {
	worker := &stubWorker{}
	enrichErr := errors.New("lookup failed")
	mw := NewEnrichMiddleware(stubEnricher{err: enrichErr})
	wrapped := mw(worker, Job{})

	err := wrapped.Run(context.Background())
	assert.ErrorIs(t, err, enrichErr)
	assert.False(t, worker.called, "wrapped worker must not run on enrichment error")
}

func TestEnrichMiddleware_PreservesKind(t *testing.T) {
	worker := &stubWorker{}
	mw := NewEnrichMiddleware(stubEnricher{tag: "enriched"})
	wrapped := mw(worker, Job{})
	assert.Equal(t, "stub", wrapped.Kind(), "Kind should pass through to the wrapped worker")
}
