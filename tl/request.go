package tl

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
	"github.com/jlaffaye/ftp"
)

func downloadHTTP(ctx context.Context, ustr string, fn string, secret Secret, auth FeedAuthorization) error {
	w, err := os.Create(fn)
	if err != nil {
		return errors.New("could not open file for writing")
	}
	defer w.Close()
	// Download HTTP
	req, err := http.NewRequest("GET", ustr, nil)
	if err != nil {
		return errors.New("invalid request")
	}
	if auth.Type == "basic_auth" {
		req.SetBasicAuth(secret.Username, secret.Password)
	} else if auth.Type == "header" {
		req.Header.Add(auth.ParamName, secret.Key)
	}
	req = req.WithContext(ctx)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// return error directly
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("response status code: %d", resp.StatusCode)
	}
	if _, err := io.Copy(w, resp.Body); err != nil {
		return errors.New("could not write response to file")
	}
	return nil
}

func downloadFTP(ctx context.Context, ustr string, fn string, secret Secret, auth FeedAuthorization) error {
	w, err := os.Create(fn)
	if err != nil {
		return errors.New("could not open file for writing")
	}
	defer w.Close()
	// Download FTP
	u, err := url.Parse(ustr)
	if err != nil {
		return errors.New("could not parse url")
	}
	p := u.Port()
	if p == "" {
		p = "21"
	}
	c, err := ftp.Dial(fmt.Sprintf("%s:%s", u.Hostname(), p), ftp.DialWithContext(ctx))
	if err != nil {
		return errors.New("could not connect to server")
	}
	if auth.Type != "basic_auth" {
		secret.Username = "anonymous"
		secret.Password = "anonymous"
	}
	err = c.Login(secret.Username, secret.Password)
	if err != nil {
		return errors.New("could not connect to server")
	}
	r, err := c.Retr(u.Path)
	if err != nil {
		// return error directly
		return err
	}
	defer r.Close()
	if _, err := io.Copy(w, r); err != nil {
		return errors.New("could not write response to file")
	}
	return nil
}

func downloadS3(ctx context.Context, ustr string, fn string, secret Secret, auth FeedAuthorization) error {
	w, err := os.Create(fn)
	if err != nil {
		return errors.New("could not open file for writing")
	}
	defer w.Close()
	// Parse url
	s3uri, err := url.Parse(ustr)
	if err != nil {
		return err
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
	s3obj, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s3bucket),
		Key:    aws.String(s3key),
	})
	if err != nil {
		return err
	}
	defer s3obj.Body.Close()
	if _, err := io.Copy(w, s3obj.Body); err != nil {
		return err
	}
	return nil
}

// AuthenticatedRequest fetches a url using a secret and auth description. Returns temp file path or error.
// Caller is responsible for deleting the file.
func AuthenticatedRequest(address string, secret Secret, auth FeedAuthorization) (string, error) {
	u, err := url.Parse(address)
	if err != nil {
		return "", errors.New("could not parse url")
	}
	if auth.Type == "query_param" {
		v, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			return "", errors.New("could not parse query string")
		}
		v.Set(auth.ParamName, secret.Key)
		u.RawQuery = v.Encode()
	} else if auth.Type == "path_segment" {
		u.Path = strings.ReplaceAll(u.Path, "{}", secret.Key)
	}
	// Prepare file
	ustr := u.String()
	tmpfile, err := ioutil.TempFile("", "fetch")
	if err != nil {
		return "", errors.New("could not create temporary file")
	}
	tmpfilepath := tmpfile.Name()
	tmpfile.Close()
	// Download
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*600))
	defer cancel()
	var reqErr error
	log.Debug().Str("url", address).Str("tmpfile", tmpfilepath).Str("auth_type", auth.Type).Msg("download")
	switch u.Scheme {
	case "http":
		reqErr = downloadHTTP(ctx, ustr, tmpfilepath, secret, auth)
	case "https":
		reqErr = downloadHTTP(ctx, ustr, tmpfilepath, secret, auth)
	case "ftp":
		reqErr = downloadFTP(ctx, ustr, tmpfilepath, secret, auth)
	case "s3":
		reqErr = downloadS3(ctx, ustr, tmpfilepath, secret, auth)
	default:
		reqErr = errors.New("unknown handler")
	}
	return tmpfilepath, reqErr
}
