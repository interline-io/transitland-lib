package request

import (
	"bytes"
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
	Download(context.Context, string, dmfr.Secret, dmfr.FeedAuthorization) (io.ReadCloser, int, error)
}

type Uploader interface {
	Upload(context.Context, string, dmfr.Secret, io.Reader) error
}

type Presigner interface {
	CreateSignedUrl(context.Context, string, string, dmfr.Secret) (string, error)
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
	log.Debug().Str("url", req.URL).Str("auth_type", req.Auth.Type).Msg("download")
	downloader, key, err := req.newDownloader(req.URL)
	if err != nil {
		return nil, 0, err
	}
	return downloader.Download(ctx, key, req.Secret, req.Auth)
}

func (req *Request) newDownloader(ustr string) (Downloader, string, error) {
	u, err := url.Parse(req.URL)
	if err != nil {
		return nil, "", errors.New("could not parse url")
	}
	var downloader Downloader
	var reqErr error
	reqUrl := req.URL
	switch u.Scheme {
	case "http":
		downloader = Http{}
	case "https":
		downloader = Http{}
	case "ftp":
		if req.AllowFTP {
			downloader = Ftp{}
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
			downloader = Local{
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

type FetchResponse struct {
	Filename     string
	Data         []byte
	ResponseSize int
	ResponseCode int
	ResponseSHA1 string
	FetchError   error
}

// AuthenticatedRequestDownload is similar to AuthenticatedRequest but writes to a temporary file.
// Fatal errors will be returned as the error; non-fatal errors as FetchResponse.FetchError
func AuthenticatedRequestDownload(address string, opts ...RequestOption) (FetchResponse, error) {
	// Create temp file
	tmpfile, err := os.CreateTemp("", "fetch")
	if err != nil {
		return FetchResponse{}, errors.New("could not create temporary file")
	}
	defer tmpfile.Close()

	// Download
	fr, err := authenticatedRequest(tmpfile, address, opts...)
	if err != nil {
		return fr, err
	}

	// Collect data
	fr.Filename = tmpfile.Name()
	return fr, nil
}

// AuthenticatedRequest fetches a url using a secret and auth description. Returns []byte, sha1, size, response code.
// Fatal errors will be returned as the error; non-fatal errors as FetchResponse.FetchError
func AuthenticatedRequest(address string, opts ...RequestOption) (FetchResponse, error) {
	// Create buffer
	var buf bytes.Buffer

	// Download
	fr, err := authenticatedRequest(&buf, address, opts...)
	if err != nil {
		return fr, err
	}

	// Collect bytes
	fr.Data = buf.Bytes()
	return fr, nil
}

func authenticatedRequest(out io.Writer, address string, opts ...RequestOption) (FetchResponse, error) {
	// 10 minute timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*600))
	defer cancel()

	// Download
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
	fr.ResponseSize, fr.ResponseSHA1, err = copyTo(out, r, req.MaxSize)

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
