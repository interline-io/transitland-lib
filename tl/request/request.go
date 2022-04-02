package request

import (
	"bytes"
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/interline-io/transitland-lib/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/jlaffaye/ftp"
)

func DownloadHTTP(ctx context.Context, ustr string, secret tl.Secret, auth tl.FeedAuthorization) (io.ReadCloser, int, error) {
	u, err := url.Parse(ustr)
	if err != nil {
		return nil, 0, errors.New("could not parse url")
	}
	if auth.Type == "query_param" {
		v, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			return nil, 0, errors.New("could not parse query string")
		}
		v.Set(auth.ParamName, secret.Key)
		u.RawQuery = v.Encode()
	} else if auth.Type == "path_segment" {
		u.Path = strings.ReplaceAll(u.Path, "{}", secret.Key)
	}
	ustr = u.String()
	// Prepare HTTP request
	req, err := http.NewRequest("GET", ustr, nil)
	if err != nil {
		return nil, 0, errors.New("invalid request")
	}
	if auth.Type == "basic_auth" {
		req.SetBasicAuth(secret.Username, secret.Password)
	} else if auth.Type == "header" {
		req.Header.Add(auth.ParamName, secret.Key)
	}
	// Make HTTP request
	req = req.WithContext(ctx)
	client := &http.Client{}
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

func DownloadFTP(ctx context.Context, ustr string, secret tl.Secret, auth tl.FeedAuthorization) (io.ReadCloser, error) {
	// Download FTP
	u, err := url.Parse(ustr)
	if err != nil {
		return nil, errors.New("could not parse url")
	}
	p := u.Port()
	if p == "" {
		p = "21"
	}
	c, err := ftp.Dial(fmt.Sprintf("%s:%s", u.Hostname(), p), ftp.DialWithContext(ctx))
	if err != nil {
		return nil, errors.New("could not connect to server")
	}
	if auth.Type != "basic_auth" {
		secret.Username = "anonymous"
		secret.Password = "anonymous"
	}
	err = c.Login(secret.Username, secret.Password)
	if err != nil {
		return nil, errors.New("could not connect to server")
	}
	r, err := c.Retr(u.Path)
	if err != nil {
		// return error directly
		return nil, err
	}
	return r, nil
}

func DownloadS3(ctx context.Context, ustr string, secret tl.Secret, auth tl.FeedAuthorization) (io.ReadCloser, error) {
	// Parse url
	s3uri, err := url.Parse(ustr)
	if err != nil {
		return nil, errors.New("could not parse url")
	}
	// Create client
	var client *s3.Client
	if secret.AWSAccessKeyID != "" && secret.AWSSecretAccessKey != "" {
		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(secret.AWSAccessKeyID, secret.AWSSecretAccessKey, "")),
		)
		if err != nil {
			return nil, err
		}
		client = s3.NewFromConfig(cfg)
	} else {
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, err
		}
		client = s3.NewFromConfig(cfg)
	}
	// Get object
	s3bucket := s3uri.Host
	s3key := strings.TrimPrefix(s3uri.Path, "/")
	s3obj, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s3bucket),
		Key:    aws.String(s3key),
	})
	if err != nil {
		return nil, err
	}
	return s3obj.Body, nil
}

func UploadS3(ctx context.Context, ustr string, secret tl.Secret, uploadFile io.Reader) error {
	s3uri, err := url.Parse(ustr)
	if err != nil {
		return errors.New("could not parse url")
	}
	// Create client
	var client *s3.Client
	if secret.AWSAccessKeyID != "" && secret.AWSSecretAccessKey != "" {
		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(secret.AWSAccessKeyID, secret.AWSSecretAccessKey, "")),
		)
		if err != nil {
			return err
		}
		client = s3.NewFromConfig(cfg)
	} else {
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return err
		}
		client = s3.NewFromConfig(cfg)
	}
	// Save object
	s3bucket := s3uri.Host
	s3key := strings.TrimPrefix(s3uri.Path, "/")
	result, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s3bucket),
		Key:    aws.String(s3key),
		Body:   uploadFile,
	})
	_ = result
	return err
}

type Request struct {
	URL        string
	AllowFTP   bool
	AllowS3    bool
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
	var r io.ReadCloser
	var reqErr error
	switch u.Scheme {
	case "http":
		r, rd, reqErr = DownloadHTTP(ctx, req.URL, req.Secret, req.Auth)
	case "https":
		r, rd, reqErr = DownloadHTTP(ctx, req.URL, req.Secret, req.Auth)
	case "ftp":
		if req.AllowFTP {
			r, reqErr = DownloadFTP(ctx, req.URL, req.Secret, req.Auth)
		} else {
			reqErr = errors.New("request not configured to allow ftp")
		}
	case "s3":
		if req.AllowS3 {
			r, reqErr = DownloadS3(ctx, req.URL, req.Secret, req.Auth)
		} else {
			reqErr = errors.New("request not configured to allow s3")
		}
	default:
		// file:// handler
		if req.AllowLocal {
			r, reqErr = os.Open(strings.TrimPrefix(req.URL, "file://"))
		} else {
			reqErr = errors.New("request not configured to allow filesystem access")
		}
	}
	return r, rd, reqErr
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

func AuthenticatedDownload2(address string, opts ...RequestOption) (FetchResponse, error) {
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

// AuthenticatedRequestDownload fetches a url using a secret and auth description. Returns temp file path, sha1, size, response code.
// Caller is responsible for deleting the file.
func AuthenticatedRequestDownload(address string, opts ...RequestOption) (string, string, int, int, error) {
	// 10 minute timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*600))
	defer cancel()
	//
	tmpfile, err := ioutil.TempFile("", "fetch")
	if err != nil {
		return "", "", 0, 0, errors.New("could not create temporary file")
	}
	tmpfilepath := tmpfile.Name()
	defer tmpfile.Close()
	req := NewRequest(address, opts...)
	r, responseCode, err := req.Request(ctx)
	if err != nil {
		return "", "", 0, responseCode, err
	}
	defer r.Close()
	responseSize, responseSha1, err := copyTo(tmpfile, r)
	fmt.Println("tmpfile:", tmpfilepath, "size:", responseSize, "sha1:", responseSha1, "code:", responseCode)
	return tmpfilepath, responseSha1, responseSize, responseCode, err
}

// AuthenticatedRequest fetches a url using a secret and auth description. Returns []byte, sha1, size, response code.
func AuthenticatedRequest(address string, opts ...RequestOption) ([]byte, string, int, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*600))
	defer cancel()
	req := NewRequest(address, opts...)
	r, responseCode, err := req.Request(ctx)
	if err != nil {
		return nil, "", 0, 0, err
	}
	defer r.Close()
	var buf bytes.Buffer
	responseSize, responseSha1, err := copyTo(&buf, r)
	return buf.Bytes(), responseSha1, responseSize, responseCode, err
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
