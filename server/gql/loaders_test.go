package gql

import (
	"context"
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
		loader := withWaitAndCapacity(time.Hour, 2, func(_ context.Context, keys []int) ([]string, []error) {
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
