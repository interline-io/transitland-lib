package request

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/stretchr/testify/assert"
)

func TestParseRetryAfter(t *testing.T) {
	testcases := []struct {
		name     string
		value    string
		expected time.Duration
	}{
		{
			name:     "empty string",
			value:    "",
			expected: 0,
		},
		{
			name:     "valid seconds",
			value:    "120",
			expected: 120 * time.Second,
		},
		{
			name:     "zero seconds",
			value:    "0",
			expected: 0,
		},
		{
			name:     "negative seconds",
			value:    "-10",
			expected: 0,
		},
		{
			name:     "invalid string",
			value:    "abc",
			expected: 0,
		},
		{
			name:     "invalid date format",
			value:    "2024-01-01",
			expected: 0,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseRetryAfter(tc.value)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	// Test with a future date (RFC 1123 format)
	futureTime := time.Now().Add(60 * time.Second).UTC().Format(http.TimeFormat)
	result := parseRetryAfter(futureTime)
	// Should be approximately 60 seconds (allow some tolerance)
	assert.True(t, result > 55*time.Second && result <= 61*time.Second,
		"expected ~60s, got %v", result)

	// Test with a past date - should return 0
	pastTime := time.Now().Add(-60 * time.Second).UTC().Format(http.TimeFormat)
	result = parseRetryAfter(pastTime)
	assert.Equal(t, time.Duration(0), result, "past date should return 0")
}

func TestHttp_calculateBackoff(t *testing.T) {
	h := &Http{
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
	}

	testcases := []struct {
		name       string
		attempt    int
		retryAfter time.Duration
		minBackoff time.Duration
		maxBackoff time.Duration
	}{
		{
			name:       "attempt 0 without retry-after",
			attempt:    0,
			retryAfter: 0,
			minBackoff: 1 * time.Second,
			maxBackoff: 2 * time.Second, // 1s + up to 25% jitter
		},
		{
			name:       "attempt 1 without retry-after",
			attempt:    1,
			retryAfter: 0,
			minBackoff: 2 * time.Second,
			maxBackoff: 3 * time.Second, // 2s + up to 25% jitter
		},
		{
			name:       "attempt 2 without retry-after",
			attempt:    2,
			retryAfter: 0,
			minBackoff: 4 * time.Second,
			maxBackoff: 5 * time.Second, // 4s + up to 25% jitter
		},
		{
			name:       "with retry-after header",
			attempt:    0,
			retryAfter: 5 * time.Second,
			minBackoff: 5 * time.Second,
			maxBackoff: 5 * time.Second, // No jitter when using Retry-After
		},
		{
			name:       "retry-after exceeds max backoff",
			attempt:    0,
			retryAfter: 60 * time.Second,
			minBackoff: 30 * time.Second, // Capped at MaxBackoff
			maxBackoff: 30 * time.Second,
		},
		{
			name:       "high attempt capped at max backoff",
			attempt:    10,
			retryAfter: 0,
			minBackoff: 30 * time.Second, // Capped at MaxBackoff
			maxBackoff: 38 * time.Second, // 30s + up to 25% jitter
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			result := h.calculateBackoff(tc.attempt, tc.retryAfter)
			assert.True(t, result >= tc.minBackoff,
				"backoff %v should be >= %v", result, tc.minBackoff)
			assert.True(t, result <= tc.maxBackoff,
				"backoff %v should be <= %v", result, tc.maxBackoff)
		})
	}
}

func TestHttp_calculateBackoff_OverflowProtection(t *testing.T) {
	h := &Http{
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     24 * time.Hour, // Very high max to test overflow protection
	}

	// High attempt number that would overflow without protection
	result := h.calculateBackoff(100, 0)
	// Should not panic and should return a reasonable value
	assert.True(t, result > 0, "backoff should be positive")
	// Backoff can include up to 25% jitter on top of maxBackoff
	assert.True(t, result <= 24*time.Hour+6*time.Hour, "backoff should be capped at max + jitter")
}

func TestHttp_calculateBackoff_SmallBackoff(t *testing.T) {
	h := &Http{
		InitialBackoff: 1 * time.Nanosecond, // Very small to test jitter protection
		MaxBackoff:     1 * time.Second,
	}

	// Should not panic when backoff/4 == 0
	result := h.calculateBackoff(0, 0)
	assert.True(t, result >= 1*time.Nanosecond, "backoff should be at least initial")
}

func TestHttp_DownloadAuth_429Retry(t *testing.T) {
	var requestCount int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count < 3 {
			// Return 429 for first two requests
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		// Success on third request
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer ts.Close()

	h := &Http{
		MaxRetries:     5,
		InitialBackoff: 10 * time.Millisecond, // Short for testing
		MaxBackoff:     100 * time.Millisecond,
	}

	ctx := context.Background()
	body, statusCode, err := h.DownloadAuth(ctx, ts.URL, dmfr.FeedAuthorization{})

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, statusCode)
	assert.NotNil(t, body)
	if body != nil {
		body.Close()
	}
	assert.Equal(t, int32(3), atomic.LoadInt32(&requestCount), "should have made 3 requests")
}

func TestHttp_DownloadAuth_429MaxRetriesExceeded(t *testing.T) {
	var requestCount int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		// Always return 429
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer ts.Close()

	h := &Http{
		MaxRetries:     2,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
	}

	ctx := context.Background()
	body, statusCode, err := h.DownloadAuth(ctx, ts.URL, dmfr.FeedAuthorization{})

	assert.Error(t, err)
	assert.Equal(t, http.StatusTooManyRequests, statusCode)
	assert.Nil(t, body)
	// Should have made maxRetries + 1 attempts (initial + retries)
	assert.Equal(t, int32(3), atomic.LoadInt32(&requestCount), "should have made 3 requests")
}

func TestHttp_DownloadAuth_429WithRetryAfterHeader(t *testing.T) {
	var requestCount int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count < 2 {
			// Return 429 with Retry-After header
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer ts.Close()

	h := &Http{
		MaxRetries:     5,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     5 * time.Second,
	}

	ctx := context.Background()
	start := time.Now()
	body, statusCode, err := h.DownloadAuth(ctx, ts.URL, dmfr.FeedAuthorization{})
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, statusCode)
	assert.NotNil(t, body)
	if body != nil {
		body.Close()
	}
	// Should have waited approximately 1 second due to Retry-After header
	assert.True(t, elapsed >= 900*time.Millisecond, "should have waited for Retry-After duration")
}

func TestHttp_DownloadAuth_ContextCancellation(t *testing.T) {
	var requestCount int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer ts.Close()

	h := &Http{
		MaxRetries:     5,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	body, _, err := h.DownloadAuth(ctx, ts.URL, dmfr.FeedAuthorization{})

	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Nil(t, body)
}

func TestHttp_DownloadAuth_NonRetryableError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	h := &Http{
		MaxRetries:     5,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
	}

	ctx := context.Background()
	body, statusCode, err := h.DownloadAuth(ctx, ts.URL, dmfr.FeedAuthorization{})

	assert.Error(t, err)
	assert.Equal(t, http.StatusNotFound, statusCode)
	assert.Nil(t, body)
}

func TestHttp_DefaultValues(t *testing.T) {
	h := &Http{}

	assert.Equal(t, defaultMaxRetries, h.getMaxRetries())
	assert.Equal(t, defaultInitialBackoff, h.getInitialBackoff())
	assert.Equal(t, defaultMaxBackoff, h.getMaxBackoff())
}

func TestHttp_CustomValues(t *testing.T) {
	h := &Http{
		MaxRetries:     10,
		InitialBackoff: 5 * time.Second,
		MaxBackoff:     10 * time.Minute,
	}

	assert.Equal(t, 10, h.getMaxRetries())
	assert.Equal(t, 5*time.Second, h.getInitialBackoff())
	assert.Equal(t, 10*time.Minute, h.getMaxBackoff())
}
