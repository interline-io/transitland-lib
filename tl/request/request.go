package request

import (
	"bytes"
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"time"

	"github.com/interline-io/transitland-lib/log"
	"github.com/interline-io/transitland-lib/tl"
)

type Downloader interface {
	Download(context.Context, string, tl.Secret, tl.FeedAuthorization) (io.ReadCloser, int, error)
}

type Uploader interface {
	Upload(context.Context, string, tl.Secret, io.Reader) error
}

type Request struct {
	URL        string
	AllowFTP   bool
	AllowS3    bool
	AllowAz    bool
	AllowLocal bool
	Secret     tl.Secret
	Auth       tl.FeedAuthorization
}

func (req *Request) Request(ctx context.Context) (io.ReadCloser, int, error) {
	rd := 0
	u, err := url.Parse(req.URL)
	if err != nil {
		return nil, rd, errors.New("could not parse url")
	}
	// Download
	log.Debug().Str("url", req.URL).Str("auth_type", req.Auth.Type).Msg("download")
	var downloader Downloader
	var reqErr error
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
			downloader = S3{}
		} else {
			reqErr = errors.New("request not configured to allow s3")
		}
	case "az":
		if req.AllowAz {
			downloader = Az{}
		} else {
			reqErr = errors.New("request not configured to allow azure")
		}
	default:
		// file:// handler
		if req.AllowLocal {
			downloader = Local{}
		} else {
			reqErr = errors.New("request not configured to allow filesystem access")
		}
	}
	if reqErr != nil {
		return nil, 0, reqErr
	}
	return downloader.Download(ctx, req.URL, req.Secret, req.Auth)
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

func WithAllowS3(req *Request) {
	req.AllowS3 = true
}

func WithAllowAz(req *Request) {
	req.AllowAz = true
}

func WithAllowLocal(req *Request) {
	req.AllowLocal = true
}

func WithAuth(secret tl.Secret, auth tl.FeedAuthorization) func(req *Request) {
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
func AuthenticatedRequestDownload(address string, opts ...RequestOption) (FetchResponse, error) {
	fr := FetchResponse{}
	// 10 minute timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*600))
	defer cancel()
	// Create temp file
	tmpfile, err := ioutil.TempFile("", "fetch")
	if err != nil {
		return fr, errors.New("could not create temporary file")
	}
	fr.Filename = tmpfile.Name()
	defer tmpfile.Close()
	// Download
	req := NewRequest(address, opts...)
	var r io.ReadCloser
	r, fr.ResponseCode, fr.FetchError = req.Request(ctx)
	if fr.FetchError != nil {
		return fr, nil
	}
	fr.ResponseSize, fr.ResponseSHA1, err = copyTo(tmpfile, r)
	if err != nil {
		return fr, err
	}
	if r != nil {
		r.Close()
	}
	return fr, nil
}

// AuthenticatedRequest fetches a url using a secret and auth description. Returns []byte, sha1, size, response code.
func AuthenticatedRequest(address string, opts ...RequestOption) (FetchResponse, error) {
	fr := FetchResponse{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*600))
	defer cancel()
	req := NewRequest(address, opts...)
	var err error
	var r io.ReadCloser
	r, fr.ResponseCode, fr.FetchError = req.Request(ctx)
	if fr.FetchError != nil {
		return fr, err
	}
	defer r.Close()
	var buf bytes.Buffer
	fr.ResponseSize, fr.ResponseSHA1, err = copyTo(&buf, r)
	fr.Data = buf.Bytes()
	return fr, err
}

func copyTo(dst io.Writer, src io.Reader) (int, string, error) {
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
		h.Write(buf[:n])
		if _, err := dst.Write(buf[:n]); err != nil {
			return 0, "", err
		}
	}
	return size, fmt.Sprintf("%x", h.Sum(nil)), nil
}
