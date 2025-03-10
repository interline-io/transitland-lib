package request

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
)

type Downloader interface {
	Download(context.Context, string) (io.ReadCloser, int, error)
	DownloadAuth(context.Context, string, dmfr.FeedAuthorization) (io.ReadCloser, int, error)
}

type Uploader interface {
	Upload(context.Context, string, io.Reader) error
}

type FetchResponse struct {
	ResponseSize   int
	ResponseCode   int
	ResponseTimeMs int
	ResponseTtfbMs int
	ResponseSHA1   string
	FetchError     error
}

type Request struct {
	URL        string
	AllowFTP   bool
	AllowLocal bool
	AllowS3    bool
	MaxSize    uint64
	Secret     dmfr.Secret
	Auth       dmfr.FeedAuthorization
}

func (req *Request) Request(ctx context.Context) (io.ReadCloser, int, error) {
	// Download
	log.For(ctx).Debug().Str("url", req.URL).Str("auth_type", req.Auth.Type).Msg("download")
	downloader, key, err := req.newDownloader(req.URL)
	if err != nil {
		return nil, 0, err
	}
	if a, ok := downloader.(CanSetSecret); ok {
		a.SetSecret(req.Secret)
	}
	return downloader.DownloadAuth(ctx, key, req.Auth)
}

func (req *Request) newDownloader(ustr string) (Downloader, string, error) {
	u, err := url.Parse(ustr)
	if err != nil {
		return nil, "", errors.New("could not parse url")
	}
	var downloader Downloader
	var reqErr error
	reqUrl := req.URL
	switch u.Scheme {
	case "http":
		downloader = &Http{}
	case "https":
		downloader = &Http{}
	case "ftp":
		if req.AllowFTP {
			downloader = &Ftp{}
		} else {
			reqErr = errors.New("request not configured to allow ftp")
		}
	case "s3":
		if req.AllowS3 {
			// Setup the S3 downloader
			downloader, reqErr = NewS3FromUrl(fmt.Sprintf("s3://%s", u.Host))
			reqUrl = u.Path
		} else {
			reqErr = errors.New("request not configured to allow s3")
		}
	default:
		if req.AllowLocal {
			// Setup the local reader
			reqDir := ""
			reqDir, reqUrl = filepath.Split(strings.TrimPrefix(req.URL, "file://"))
			downloader = &Local{
				Directory: reqDir,
			}
		} else {
			reqErr = errors.New("request not configured to allow filesystem access")
		}
	}
	return downloader, reqUrl, reqErr
}

func NewRequest(address string, opts ...RequestOption) *Request {
	req := &Request{URL: address}
	for _, opt := range opts {
		opt(req)
	}
	return req
}

type RequestOption func(*Request)

func WithAllowFTP(req *Request) {
	req.AllowFTP = true
}

func WithAllowLocal(req *Request) {
	req.AllowLocal = true
}

func WithAllowS3(req *Request) {
	req.AllowS3 = true
}

func WithMaxSize(s uint64) RequestOption {
	return func(req *Request) {
		req.MaxSize = s
	}
}

func WithAuth(secret dmfr.Secret, auth dmfr.FeedAuthorization) func(req *Request) {
	return func(req *Request) {
		req.Secret = secret
		req.Auth = auth
	}
}

// AuthenticatedRequestDownload is similar to AuthenticatedRequest but writes to a temporary file.
// Fatal errors will be returned as the error; non-fatal errors as FetchResponse.FetchError
func AuthenticatedRequestDownload(ctx context.Context, address string, opts ...RequestOption) (string, FetchResponse, error) {
	// Create temp file
	tmpfile, err := os.CreateTemp("", "fetch")
	if err != nil {
		return "", FetchResponse{}, errors.New("could not create temporary file")
	}
	defer tmpfile.Close()

	// Download
	fr, err := AuthenticatedRequest(ctx, tmpfile, address, opts...)
	if err != nil {
		return "", fr, err
	}

	// Collect data
	return tmpfile.Name(), fr, nil
}

// AuthenticatedRequestContext fetches a url using a secret and auth description.
func AuthenticatedRequest(ctx context.Context, out io.Writer, address string, opts ...RequestOption) (FetchResponse, error) {
	// 10 minute timeout
	ctx, cancel := context.WithTimeout(ctx, time.Duration(time.Second*600))
	defer cancel()

	// Create request and wait for response
	t := time.Now()
	fr := FetchResponse{}
	req := NewRequest(address, opts...)
	var r io.ReadCloser
	r, fr.ResponseCode, fr.FetchError = req.Request(ctx)
	if fr.FetchError != nil {
		return fr, nil
	}
	defer r.Close()

	// Write response
	var err error
	fr.ResponseTtfbMs = int(time.Since(t) / time.Millisecond)
	fr.ResponseSize, fr.ResponseSHA1, err = copyTo(out, r, req.MaxSize)
	fr.ResponseTimeMs = int(time.Since(t) / time.Millisecond)

	// Check for canceled
	if ctxErr := ctx.Err(); ctxErr != nil {
		// Return a non fatal error
		fr.FetchError = ctxErr
		return fr, nil
	} else if err != nil {
		// Return a fatal error
		fr.FetchError = err
		return fr, err
	}
	return fr, nil
}

func copyTo(dst io.Writer, src io.Reader, maxSize uint64) (int, string, error) {
	size := 0
	h := sha1.New()
	buf := make([]byte, 1024*1024)
	for {
		n, err := src.Read(buf)
		if err != nil && err != io.EOF {
			return 0, "", err
		}
		if n == 0 {
			break
		}
		size += n
		if maxSize > 0 && size > int(maxSize) {
			return 0, "", errors.New("exceeded max size")
		}
		h.Write(buf[:n])
		if _, err := dst.Write(buf[:n]); err != nil {
			return 0, "", err
		}
	}
	return size, fmt.Sprintf("%x", h.Sum(nil)), nil
}
