package gql

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWithWaitAndCapacity_Errors(t *testing.T) {
	ctx := context.Background()
	t.Run("failed batch", func(t *testing.T) {
		// A failed batch returns no results and an error for each key. Capacity 2
		// flushes on the second load, so the two keys run in one batch.
		loader := withWaitAndCapacity(time.Second, 2, func(_ context.Context, keys []int) ([]string, []error) {
			errs := make([]error, len(keys))
			for i, key := range keys {
				errs[i] = fmt.Errorf("key %d failed", key)
			}
			return nil, errs
		})
		first, second := loader.Load(ctx, 1), loader.Load(ctx, 2)
		_, err := first()
		assert.EqualError(t, err, "key 1 failed")
		_, err = second()
		assert.EqualError(t, err, "key 2 failed")
	})
	t.Run("missing results", func(t *testing.T) {
		// Too few results and no errors is a finder bug, and fails rather than
		// answering with zero values.
		loader := withWaitAndCapacity(waitTime, maxBatch, func(_ context.Context, keys []int) ([]string, []error) {
			return nil, nil
		})
		_, err := loader.Load(ctx, 1)()
		assert.Error(t, err)
	})
}

func Test_batchShapeOK(t *testing.T) {
	err := errors.New("failed")
	tcs := []struct {
		name    string
		results int
		errs    []error
		ok      bool
	}{
		{"results", 3, nil, true},
		{"results with an error for each key", 3, []error{nil, err, nil}, true},
		{"failed batch", 0, []error{err, err, err}, true},
		{"no results and no errors", 0, nil, false},
		{"failed batch with a nil error", 0, []error{err, nil, err}, false},
		{"too few results", 2, []error{err, err, err}, false},
		{"results with one error", 3, []error{err}, false},
		{"too many results", 4, nil, false},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.ok, batchShapeOK(3, tc.results, tc.errs))
		})
	}
}
