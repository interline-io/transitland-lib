package kvcache

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// A caller that missed the local tier just before another flight installed the
// key takes that flight's value instead of refreshing again.
func TestCache_FlightRechecksLocal(t *testing.T) {
	ctx := context.Background()
	calls := 0
	c := NewRefreshCache[string, string](nil, "test", func(ctx context.Context, key string) (string, error) {
		calls++
		return "refreshed", nil
	})
	if err := c.Set(ctx, "k", "held"); err != nil {
		t.Fatal(err)
	}
	it, ok := c.loadOrRefresh(ctx, "k")
	assert.True(t, ok)
	assert.Equal(t, "held", it.Value)
	assert.Equal(t, 0, calls)
}
