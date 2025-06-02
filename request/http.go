package request

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/interline-io/log"
	tl "github.com/interline-io/transitland-lib"
	"github.com/interline-io/transitland-lib/dmfr"
)

func init() {
	var _ Downloader = &Http{}
}

type Http struct {
	secret dmfr.Secret
}

func (r *Http) SetSecret(secret dmfr.Secret) error {
	r.secret = secret
	return nil
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
	if auth.Type == "query_param" {
		v, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			return nil, 0, errors.New("could not parse query string")
		}
		v.Set(auth.ParamName, r.secret.Key)
		u.RawQuery = v.Encode()
	} else if auth.Type == "path_segment" {
		u.Path = strings.ReplaceAll(u.Path, "{}", r.secret.Key)
	} else if auth.Type == "replace_url" {
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
	if auth.Type == "basic_auth" {
		req.SetBasicAuth(r.secret.Username, r.secret.Password)
	} else if auth.Type == "header" {
		req.Header.Add(auth.ParamName, r.secret.Key)
	}

	// Make HTTP request
	req.Header.Set("User-Agent", fmt.Sprintf("transitland/%s", tl.Version.Tag))
	// If the following headers are not set, some CDNs may block the request as coming from a bot rather than a browser
	req.Header.Set("Accept", "application/zip,application/x-zip-compressed,application/octet-stream;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "")

	// Log request details
	log.For(ctx).Trace().
		Str("url", ustr).
		Str("method", req.Method).
		Str("host", req.Host).
		Str("user_agent", req.UserAgent()).
		Interface("headers", req.Header).
		Msg("making HTTP request")

	// Remove default ports from host header if explicitly specified as it
	// may break pre-signed S3 URLs or other systems that rely on the host header
	removeDefaultPortFromHost(req)

	// Create custom transport with logging
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			CipherSuites: []uint16{
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			},
		},
	}

	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			removeDefaultPortFromHost(req)
			return nil
		},
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		// return error directly
		return nil, 0, err
	}
	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, resp.StatusCode, fmt.Errorf("response status code: %d", resp.StatusCode)
	}
	return resp.Body, resp.StatusCode, nil
}

// loggingConn wraps a net.Conn to log when it's closed
type loggingConn struct {
	net.Conn
	onClose func()
}

func (c *loggingConn) Close() error {
	c.onClose()
	return c.Conn.Close()
}
