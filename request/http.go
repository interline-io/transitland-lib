package request

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
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
	// Default retry configuration for HTTP 429 responses.
	// With 5 retries and exponential backoff starting at 10s:
	//   Attempt 1: immediate
	//   Attempt 2: after ~10s  (10s + jitter)
	//   Attempt 3: after ~20s  (20s + jitter)
	//   Attempt 4: after ~40s  (40s + jitter)
	//   Attempt 5: after ~80s  (80s + jitter)
	//   Attempt 6: after ~160s (160s + jitter)
	// Total max wait time: ~310s (~5 minutes)
	defaultMaxRetries     = 5
	defaultInitialBackoff = 10 * time.Second
	defaultMaxBackoff     = 5 * time.Minute
)

type Http struct {
	secret         dmfr.Secret
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
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

func (r *Http) getInitialBackoff() time.Duration {
	if r.InitialBackoff > 0 {
		return r.InitialBackoff
	}
	return defaultInitialBackoff
}

func (r *Http) getMaxBackoff() time.Duration {
	if r.MaxBackoff > 0 {
		return r.MaxBackoff
	}
	return defaultMaxBackoff
}

// parseRetryAfter parses the Retry-After header value.
// It can be either a number of seconds or an HTTP-date.
func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	// Try parsing as seconds first
	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second
	}
	// Try parsing as HTTP-date (RFC 1123)
	if t, err := time.Parse(time.RFC1123, value); err == nil {
		return time.Until(t)
	}
	return 0
}

// calculateBackoff returns the backoff duration for the given attempt.
// It uses exponential backoff with jitter.
func (r *Http) calculateBackoff(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		// Use Retry-After header if provided, but cap at maxBackoff
		maxBackoff := r.getMaxBackoff()
		if retryAfter > maxBackoff {
			return maxBackoff
		}
		return retryAfter
	}
	// Exponential backoff: initialBackoff * 2^attempt
	backoff := r.getInitialBackoff() * (1 << attempt)
	maxBackoff := r.getMaxBackoff()
	if backoff > maxBackoff {
		backoff = maxBackoff
	}
	// Add jitter: random value between 0 and 25% of backoff
	jitter := time.Duration(rand.Int63n(int64(backoff / 4)))
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
			return resp.Body, resp.StatusCode, nil
		}

		// Handle rate limiting (429 Too Many Requests)
		if resp.StatusCode == http.StatusTooManyRequests && attempt < maxRetries {
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
			backoff := r.calculateBackoff(attempt, retryAfter)

			log.Info().
				Str("url", ustr).
				Int("attempt", attempt+1).
				Int("max_retries", maxRetries).
				Dur("backoff", backoff).
				Str("retry_after", resp.Header.Get("Retry-After")).
				Msg("rate limited, retrying")

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
