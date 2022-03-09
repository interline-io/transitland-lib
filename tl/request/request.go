package request

import (
	"context"
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

func DownloadHTTP(ctx context.Context, ustr string, secret tl.Secret, auth tl.FeedAuthorization) (io.ReadCloser, error) {
	u, err := url.Parse(ustr)
	if err != nil {
		return nil, errors.New("could not parse url")
	}
	if auth.Type == "query_param" {
		v, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			return nil, errors.New("could not parse query string")
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
		return nil, errors.New("invalid request")
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
		return nil, err
	}
	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, fmt.Errorf("response status code: %d", resp.StatusCode)
	}
	return resp.Body, nil
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

func (req *Request) Request(ctx context.Context) (io.ReadCloser, error) {
	u, err := url.Parse(req.URL)
	if err != nil {
		return nil, errors.New("could not parse url")
	}
	// Download
	log.Debug().Str("url", req.URL).Str("auth_type", req.Auth.Type).Msg("download")
	var r io.ReadCloser
	reqErr := errors.New("unknown handler")
	switch u.Scheme {
	case "http":
		r, reqErr = DownloadHTTP(ctx, req.URL, req.Secret, req.Auth)
	case "https":
		r, reqErr = DownloadHTTP(ctx, req.URL, req.Secret, req.Auth)
	case "ftp":
		if req.AllowFTP {
			r, reqErr = DownloadFTP(ctx, req.URL, req.Secret, req.Auth)
		}
	case "s3":
		if req.AllowS3 {
			r, reqErr = DownloadS3(ctx, req.URL, req.Secret, req.Auth)
		}
	default:
		// file:// handler
		if req.AllowLocal {
			r, reqErr = os.Open(strings.TrimPrefix(req.URL, "file://"))
		}
	}
	return r, reqErr
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

// AuthenticatedRequestDownload fetches a url using a secret and auth description. Returns temp file path or error.
// Caller is responsible for deleting the file.
func AuthenticatedRequestDownload(address string, opts ...RequestOption) (string, error) {
	// 10 minute timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*600))
	defer cancel()
	//
	tmpfile, err := ioutil.TempFile("", "fetch")
	if err != nil {
		return "", errors.New("could not create temporary file")
	}
	tmpfilepath := tmpfile.Name()
	defer tmpfile.Close()
	req := NewRequest(address, opts...)
	r, err := req.Request(ctx)
	if err != nil {
		return "", err
	}
	defer r.Close()
	_, err = io.Copy(tmpfile, r)
	return tmpfilepath, err
}
