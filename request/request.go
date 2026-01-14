package request

import (
	"context"
	"crypto/md5"
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

// md5FromStream calculates MD5 hash from a stream (non-seekable reader).
// Returns nil if an error occurs during reading.
func md5FromStream(r io.Reader) []byte {
	h := md5.New()
	if _, err := io.Copy(h, r); err != nil {
		log.Info().Err(err).Msg("md5FromStream: failed to calculate MD5")
		return nil
	}
	return h.Sum(nil)
}

// md5FromReader calculates MD5 hash from a reader if it supports seeking.
// Returns nil if the reader is not seekable or if an error occurs.
// The reader position is reset to the beginning after calculation.
func md5FromReader(r io.Reader) []byte {
	seeker, ok := r.(io.ReadSeeker)
	if !ok {
		log.Trace().Msg("md5FromReader: reader is not seekable, skipping MD5 calculation")
		return nil
	}
	if _, err := seeker.Seek(0, io.SeekStart); err != nil {
		log.Info().Err(err).Msg("md5FromReader: failed to seek to start")
		return nil
	}
	result := md5FromStream(seeker)
	if result == nil {
		return nil
	}
	if _, err := seeker.Seek(0, io.SeekStart); err != nil {
		log.Error().Err(err).Msg("md5FromReader: failed to reset position after MD5 calculation")
		return nil
	}
	return result
}

// md5Equal compares two MD5 hashes for equality.
func md5Equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

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
	var expectedSize int64 = -1
	r, fr.ResponseCode, fr.FetchError = req.Request(ctx)
	if fr.FetchError != nil {
		return fr, nil
	}
	defer r.Close()

	// Get expected size from Content-Length header if available
	if httpRespReader, ok := r.(*httpResponseReader); ok && httpRespReader != nil {
		if httpRespReader.ContentLength > 0 {
			expectedSize = httpRespReader.ContentLength
		}
	}

	// Write response
	var err error
	fr.ResponseTtfbMs = int(time.Since(t) / time.Millisecond)
	fr.ResponseSize, fr.ResponseSHA1, err = copyToWithSizeCheck(out, r, req.MaxSize, expectedSize)
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

// copyToWithSizeCheck copies from src to dst while computing SHA1 hash and checking size limits.
// It provides limited integrity checking for truncated downloads only when the expected size is known
// (e.g., from HTTP Content-Length header). This check is not comprehensive: servers may not set
// Content-Length, may set it incorrectly, or use chunked transfer encoding. The SHA1 hash returned
// provides the primary integrity verification mechanism for comparing subsequent fetches.
func copyToWithSizeCheck(dst io.Writer, src io.Reader, maxSize uint64, expectedSize int64) (int, string, error) {
	size := 0
	h := sha1.New()
	buf := make([]byte, 1024*1024)
	for {
		n, readErr := src.Read(buf)
		// Process any data that was read, even if there's also an error
		if n > 0 {
			size += n
			if maxSize > 0 && size > int(maxSize) {
				return 0, "", fmt.Errorf("exceeded max size at offset %d: %d > %d", size-n, size, maxSize)
			}
			h.Write(buf[:n])
			written, writeErr := dst.Write(buf[:n])
			if writeErr != nil {
				return 0, "", fmt.Errorf("write error at offset %d: %w", size-n, writeErr)
			}
			if written != n {
				return 0, "", fmt.Errorf("partial write at offset %d: wrote %d of %d bytes", size-n, written, n)
			}
		}
		// Check for end of stream
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return 0, "", fmt.Errorf("read error at offset %d: %w", size, readErr)
		}
	}
	// Verify size matches Content-Length if provided
	// Note: We log warnings for mismatches but don't fail, as some servers misconfigure Content-Length
	// However, if downloaded size is LESS than Content-Length, that indicates truncation
	if expectedSize > 0 {
		if size < int(expectedSize) {
			// Downloaded less than Content-Length - this indicates truncation, fail
			return 0, "", fmt.Errorf("download truncated: expected %d bytes (Content-Length), got %d bytes", expectedSize, size)
		} else if size > int(expectedSize) {
			// Downloaded more than Content-Length - server misconfiguration, but file may be valid
			// Log warning but allow (server may have incorrect Content-Length header)
			log.Debug().Msgf("Content-Length mismatch (server may be misconfigured): expected %d bytes, got %d bytes", expectedSize, size)
		}
	}
	return size, fmt.Sprintf("%x", h.Sum(nil)), nil
}
