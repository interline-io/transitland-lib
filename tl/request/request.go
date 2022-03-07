package request

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
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

type Secret = tl.Secret
type FeedAuthorization = tl.FeedAuthorization

func downloadHTTP(ctx context.Context, ustr string, secret Secret, auth FeedAuthorization) (io.ReadCloser, error) {
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

func downloadFTP(ctx context.Context, ustr string, secret Secret, auth FeedAuthorization) (io.ReadCloser, error) {
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

func downloadS3(ctx context.Context, ustr string, secret Secret, auth FeedAuthorization) (io.ReadCloser, error) {
	// Parse url
	s3uri, err := url.Parse(ustr)
	if err != nil {
		return nil, err
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
	// Save object
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

// AuthenticatedRequest fetches a url using a secret and auth description. Returns ReadCloser, caller responsible for closing.
func AuthenticatedRequest(address string, secret Secret, auth FeedAuthorization) (io.ReadCloser, error) {
	u, err := url.Parse(address)
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
	// Prepare file
	ustr := u.String()
	// Download
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*600))
	defer cancel()
	var r io.ReadCloser
	var reqErr error
	log.Debug().Str("url", address).Str("auth_type", auth.Type).Msg("download")
	switch u.Scheme {
	case "http":
		r, reqErr = downloadHTTP(ctx, ustr, secret, auth)
	case "https":
		r, reqErr = downloadHTTP(ctx, ustr, secret, auth)
	case "ftp":
		r, reqErr = downloadFTP(ctx, ustr, secret, auth)
	case "s3":
		r, reqErr = downloadS3(ctx, ustr, secret, auth)
	default:
		reqErr = errors.New("unknown handler")
	}
	return r, reqErr
}

// AuthenticatedRequestDownload fetches a url using a secret and auth description. Returns temp file path or error.
// Caller is responsible for deleting the file.
func AuthenticatedRequestDownload(address string, secret Secret, auth FeedAuthorization) (string, error) {
	tmpfile, err := ioutil.TempFile("", "fetch")
	if err != nil {
		return "", errors.New("could not create temporary file")
	}
	tmpfilepath := tmpfile.Name()
	tmpfile.Close()
	//
	r, err := AuthenticatedRequest(address, secret, auth)
	if err != nil {
		return "", err
	}
	defer r.Close()
	_, err = io.Copy(tmpfile, r)
	return tmpfilepath, err
}
