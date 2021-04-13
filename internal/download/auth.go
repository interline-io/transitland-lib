package download

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/jlaffaye/ftp"
)

func downloadHTTP(ustr string, fn string, secret Secret, auth tl.FeedAuthorization) error {
	w, err := os.Create(fn)
	if err != nil {
		return err
	}
	defer w.Close()
	// Download HTTP
	client := &http.Client{
		Timeout: 600 * time.Second,
	}
	req, err := http.NewRequest("GET", ustr, nil)
	if err != nil {
		return err
	}
	if auth.Type == "basic_auth" {
		req.SetBasicAuth(secret.Username, secret.Password)
	} else if auth.Type == "header" {
		req.Header.Add(auth.ParamName, secret.Key)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, resp.Body); err != nil {
		return err
	}
	return nil
}

func downloadFTP(ustr string, fn string, secret Secret, auth tl.FeedAuthorization) error {
	w, err := os.Create(fn)
	if err != nil {
		return err
	}
	defer w.Close()
	// Download FTP
	u, err := url.Parse(ustr)
	if err != nil {
		return err
	}
	p := u.Port()
	if p == "" {
		p = "21"
	}
	c, err := ftp.Dial(fmt.Sprintf("%s:%s", u.Hostname(), p), ftp.DialWithTimeout(600*time.Second))
	if err != nil {
		return err
	}
	if auth.Type != "basic_auth" {
		secret.Username = "anonymous"
		secret.Password = "anonymous"
	}
	err = c.Login(secret.Username, secret.Password)
	if err != nil {
		return err
	}
	r, err := c.Retr(u.Path)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	return nil
}

func downloadS3(ustr string, fn string, secret Secret, auth tl.FeedAuthorization) error {
	awscmd := exec.Command("aws", "s3", "cp", ustr, fn)
	if secret.AWSAccessKeyID != "" || secret.AWSSecretAccessKey != "" {
		env := []string{
			fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", secret.AWSAccessKeyID),
			fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", secret.AWSSecretAccessKey),
		}
		env = append(env, os.Environ()...)
		awscmd.Env = env
	}
	if output, err := awscmd.Output(); err != nil {
		return fmt.Errorf("error downloading %s: %s %s", ustr, output, err.Error())
	}
	return nil
}

// AuthenticatedRequest fetches a url using a secret and auth description. Returns temp file path or error.
func AuthenticatedRequest(address string, secret Secret, auth tl.FeedAuthorization) (string, error) {
	u, err := url.Parse(address)
	if err != nil {
		return "", err
	}
	if auth.Type == "query_param" {
		v, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			return "", err
		}
		v.Set(auth.ParamName, secret.Key)
		u.RawQuery = v.Encode()
	} else if auth.Type == "path_segment" {
		u.Path = strings.ReplaceAll(u.Path, "{}", secret.Key)
	}
	// prepare worker
	ch := make(chan error)
	ustr := u.String()
	tmpfile, err := ioutil.TempFile("", "fetch")
	if err != nil {
		return "", err
	}
	tmpfilepath := tmpfile.Name()
	tmpfile.Close()
	go func() {
		var err error
		if u.Scheme == "http" || u.Scheme == "https" {
			err = downloadHTTP(ustr, tmpfilepath, secret, auth)
		} else if u.Scheme == "ftp" {
			err = downloadFTP(ustr, tmpfilepath, secret, auth)
		} else if u.Scheme == "s3" {
			err = downloadS3(ustr, tmpfilepath, secret, auth)
		} else {
			err = errors.New("unknown handler")
		}
		ch <- err
	}()
	// prepare timeout
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(600 * time.Second)
		timeout <- true
	}()
	select {
	case a := <-ch:
		if a != nil {
			return "", a
		}
	case <-timeout:
		return "", errors.New("operation timed out")
	}
	return tmpfilepath, nil
}
