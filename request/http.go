package request

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/interline-io/log"
	tl "github.com/interline-io/transitland-lib"
	"github.com/interline-io/transitland-lib/dmfr"
)

func init() {
	var _ Downloader = &Http{}
}

const (
	// Default retry configuration for transient HTTP errors (429, 502, 503, 504).
	// Retry schedule:
	//   Attempt 1: immediate
	//   Attempt 2: after ~10s (10s + jitter)
	//   Attempt 3: after ~30s (30s + jitter)
	//   Attempt 4: after ~90s (90s + jitter)
	// Total max wait time: ~130s (~2 minutes)
	defaultMaxRetries = 3
)

// defaultBackoffSchedule defines the backoff duration for each retry attempt.
var defaultBackoffSchedule = []time.Duration{
	10 * time.Second,
	30 * time.Second,
	90 * time.Second,
}

type Http struct {
	secret dmfr.Secret
	// MaxRetries sets the maximum number of retry attempts for a request.
	// If MaxRetries is zero or negative, a default value (defaultMaxRetries) is used.
	MaxRetries int
	// BackoffSchedule defines the backoff duration for each retry attempt.
	// If nil or empty, defaultBackoffSchedule is used.
	BackoffSchedule []time.Duration
}

func (r *Http) SetSecret(secret dmfr.Secret) error {
	r.secret = secret
	return nil
}

func (r *Http) getMaxRetries() int {
	if r.MaxRetries > 0 {
		return r.MaxRetries
	}
	return defaultMaxRetries
}

func (r *Http) getBackoffSchedule() []time.Duration {
	if len(r.BackoffSchedule) > 0 {
		return r.BackoffSchedule
	}
	return defaultBackoffSchedule
}

// isRetryableStatus returns true for HTTP status codes that indicate
// transient errors worth retrying.
func isRetryableStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests, // 429
		http.StatusBadGateway,         // 502
		http.StatusServiceUnavailable, // 503
		http.StatusGatewayTimeout:     // 504
		return true
	}
	return false
}

// parseRetryAfter parses the Retry-After header value.
// It can be either a number of seconds or an HTTP-date.
func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	// Try parsing as seconds first
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds < 0 {
			// Per RFC 7231, delay-seconds must be a non-negative decimal integer.
			// Treat negative values as invalid and fall back to default logic.
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	// Try parsing as HTTP-date (supports RFC 1123, RFC 850, and ANSI C asctime)
	if t, err := http.ParseTime(value); err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0
		}
		return d
	}
	return 0
}

// calculateBackoff returns the backoff duration for the given attempt.
// It uses the backoff schedule with jitter, or the Retry-After header if provided.
func (r *Http) calculateBackoff(attempt int, retryAfter time.Duration) time.Duration {
	schedule := r.getBackoffSchedule()
	// Get the base backoff from schedule, capping at the last value
	var backoff time.Duration
	if attempt < len(schedule) {
		backoff = schedule[attempt]
	} else if len(schedule) > 0 {
		backoff = schedule[len(schedule)-1]
	}
	// Use Retry-After header if provided and greater than scheduled backoff
	if retryAfter > backoff {
		backoff = retryAfter
	}
	// Add jitter: random value between 0 and 25% of backoff.
	// Guard against very small backoff values that would make backoff/4 == 0
	maxJitter := backoff / 4
	if maxJitter <= 0 {
		return backoff
	}
	jitter := rand.N(maxJitter)
	return backoff + jitter
}

func removeDefaultPortFromHost(req *http.Request) {
	if (req.URL.Scheme == "https" && strings.HasSuffix(req.URL.Host, ":443")) ||
		(req.URL.Scheme == "http" && strings.HasSuffix(req.URL.Host, ":80")) {
		req.Host = strings.Split(req.URL.Host, ":")[0]
	}
}

func (r Http) Download(ctx context.Context, ustr string) (io.ReadCloser, int, error) {
	return r.DownloadAuth(ctx, ustr, dmfr.FeedAuthorization{})
}

func (r Http) DownloadAuth(ctx context.Context, ustr string, auth dmfr.FeedAuthorization) (io.ReadCloser, int, error) {
	u, err := url.Parse(ustr)
	if err != nil {
		return nil, 0, errors.New("could not parse url")
	}
	switch auth.Type {
	case "query_param":
		v, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			return nil, 0, errors.New("could not parse query string")
		}
		v.Set(auth.ParamName, r.secret.Key)
		u.RawQuery = v.Encode()
	case "path_segment":
		u.Path = strings.ReplaceAll(u.Path, "{}", r.secret.Key)
	case "replace_url":
		u, err = url.Parse(r.secret.ReplaceUrl)
		if err != nil {
			return nil, 0, errors.New("could not parse replacement query string")
		}
	}
	ustr = u.String()

	// Prepare HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", ustr, nil)
	if err != nil {
		return nil, 0, errors.New("invalid request")
	}

	// Set basic auth, if used
	switch auth.Type {
	case "basic_auth":
		req.SetBasicAuth(r.secret.Username, r.secret.Password)
	case "header":
		req.Header.Add(auth.ParamName, r.secret.Key)
	}

	// Make HTTP request
	req.Header.Set("User-Agent", fmt.Sprintf("transitland/%s", tl.Version.Tag))
	// If the following headers are not set, some CDNs may block the request as coming from a bot rather than a browser
	req.Header.Set("Accept", "application/zip,application/x-zip-compressed,application/octet-stream;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "")

	// Remove default ports from host header if explicitly specified as it
	// may break pre-signed S3 URLs or other systems that rely on the host header
	removeDefaultPortFromHost(req)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			removeDefaultPortFromHost(req)
			return nil
		},
	}

	maxRetries := r.getMaxRetries()
	var lastErr error
	var lastStatusCode int

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Clone the request for retry (body is nil for GET, so this is safe)
		reqCopy := req.Clone(ctx)

		resp, err := client.Do(reqCopy)
		if err != nil {
			// Network error - return immediately
			return nil, 0, err
		}

		// Success
		if resp.StatusCode < 400 {
			// Wrap response body to preserve Content-Length for verification
			return &httpResponseReader{
				ReadCloser:    resp.Body,
				ContentLength: resp.ContentLength,
			}, resp.StatusCode, nil
		}

		// Handle retryable errors (429, 502, 503, 504)
		if isRetryableStatus(resp.StatusCode) && attempt < maxRetries {
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
			backoff := r.calculateBackoff(attempt, retryAfter)

			log.Info().
				Str("url", ustr).
				Int("status_code", resp.StatusCode).
				Int("attempt", attempt+1).
				Int("max_retries", maxRetries).
				Dur("backoff", backoff).
				Str("retry_after", resp.Header.Get("Retry-After")).
				Msg("transient error, retrying")

			resp.Body.Close()

			// Wait for backoff duration or until context is cancelled
			select {
			case <-ctx.Done():
				return nil, 0, ctx.Err()
			case <-time.After(backoff):
				// Continue to next retry attempt
			}
			continue
		}

		// Non-retryable error or max retries exceeded
		lastErr = fmt.Errorf("response status code: %d", resp.StatusCode)
		lastStatusCode = resp.StatusCode
		resp.Body.Close()
		break
	}

	return nil, lastStatusCode, lastErr
}

// httpResponseReader wraps http.Response.Body to preserve Content-Length
type httpResponseReader struct {
	io.ReadCloser
	ContentLength int64
}
