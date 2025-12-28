package request

import (
	"context"
	"fmt"
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
		BackoffSchedule: []time.Duration{
			1 * time.Second,
			3 * time.Second,
			9 * time.Second,
		},
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
			maxBackoff: 1250 * time.Millisecond, // 1s + up to 25% jitter
		},
		{
			name:       "attempt 1 without retry-after",
			attempt:    1,
			retryAfter: 0,
			minBackoff: 3 * time.Second,
			maxBackoff: 3750 * time.Millisecond, // 3s + up to 25% jitter
		},
		{
			name:       "attempt 2 without retry-after",
			attempt:    2,
			retryAfter: 0,
			minBackoff: 9 * time.Second,
			maxBackoff: 11250 * time.Millisecond, // 9s + up to 25% jitter
		},
		{
			name:       "attempt beyond schedule uses last value",
			attempt:    10,
			retryAfter: 0,
			minBackoff: 9 * time.Second,
			maxBackoff: 11250 * time.Millisecond, // 9s + up to 25% jitter
		},
		{
			name:       "retry-after greater than scheduled backoff",
			attempt:    0,
			retryAfter: 5 * time.Second,
			minBackoff: 5 * time.Second,
			maxBackoff: 6250 * time.Millisecond, // 5s + up to 25% jitter
		},
		{
			name:       "retry-after less than scheduled backoff uses schedule",
			attempt:    2,
			retryAfter: 2 * time.Second,
			minBackoff: 9 * time.Second,
			maxBackoff: 11250 * time.Millisecond, // 9s + up to 25% jitter
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

func TestHttp_calculateBackoff_SmallBackoff(t *testing.T) {
	h := &Http{
		BackoffSchedule: []time.Duration{1 * time.Nanosecond}, // Very small to test jitter protection
	}

	// Should not panic when backoff/4 == 0
	result := h.calculateBackoff(0, 0)
	assert.True(t, result >= 1*time.Nanosecond, "backoff should be at least initial")
}

func TestIsRetryableStatus(t *testing.T) {
	testcases := []struct {
		statusCode int
		retryable  bool
	}{
		{200, false},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{429, true},  // Too Many Requests
		{500, false}, // Internal Server Error - not retryable
		{502, true},  // Bad Gateway
		{503, true},  // Service Unavailable
		{504, true},  // Gateway Timeout
	}

	for _, tc := range testcases {
		t.Run(fmt.Sprintf("status_%d", tc.statusCode), func(t *testing.T) {
			assert.Equal(t, tc.retryable, isRetryableStatus(tc.statusCode))
		})
	}
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
		MaxRetries:      3,
		BackoffSchedule: []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 30 * time.Millisecond},
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

func TestHttp_DownloadAuth_503Retry(t *testing.T) {
	var requestCount int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count < 2 {
			// Return 503 for first request
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		// Success on second request
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer ts.Close()

	h := &Http{
		MaxRetries:      3,
		BackoffSchedule: []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 30 * time.Millisecond},
	}

	ctx := context.Background()
	body, statusCode, err := h.DownloadAuth(ctx, ts.URL, dmfr.FeedAuthorization{})

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, statusCode)
	assert.NotNil(t, body)
	if body != nil {
		body.Close()
	}
	assert.Equal(t, int32(2), atomic.LoadInt32(&requestCount), "should have made 2 requests")
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
		MaxRetries:      2,
		BackoffSchedule: []time.Duration{10 * time.Millisecond, 20 * time.Millisecond},
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
		MaxRetries:      3,
		BackoffSchedule: []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 30 * time.Millisecond},
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
		MaxRetries:      3,
		BackoffSchedule: []time.Duration{1 * time.Second, 2 * time.Second, 3 * time.Second},
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
		MaxRetries:      3,
		BackoffSchedule: []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 30 * time.Millisecond},
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
	assert.Equal(t, defaultBackoffSchedule, h.getBackoffSchedule())
}

func TestHttp_CustomValues(t *testing.T) {
	customSchedule := []time.Duration{5 * time.Second, 15 * time.Second, 45 * time.Second}
	h := &Http{
		MaxRetries:      10,
		BackoffSchedule: customSchedule,
	}

	assert.Equal(t, 10, h.getMaxRetries())
	assert.Equal(t, customSchedule, h.getBackoffSchedule())
}
