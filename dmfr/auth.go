package dmfr

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/internal/log"
	"github.com/jlaffaye/ftp"
)

// AuthenticatedRequest fetches a url using a secret and auth description.
func AuthenticatedRequest(address string, secret Secret, auth gotransit.FeedAuthorization) (string, error) {
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
		// log.Debug("Using query_param authentication: %s = %s", auth.ParamName, secret.Key)
	} else if auth.Type == "path_segment" {
		u.Path = strings.ReplaceAll(u.Path, "{}", secret.Key)
		// log.Debug("Using path_segment authentication: %s -> %s", p, u.Path)
	}
	ustr := u.String()
	// Get the temporary file and full path
	tmpfile, err := ioutil.TempFile("", "fetch")
	if err != nil {
		return "", err
	}
	defer tmpfile.Close()
	tmpfilepath := tmpfile.Name()
	log.Debug("AuthorizedRequest downloading %s -> %s", address, tmpfilepath)
	if u.Scheme == "http" {
		// Download HTTP
		client := &http.Client{}
		req, err := http.NewRequest("GET", ustr, nil)
		if err != nil {
			return "", err
		}
		if auth.Type == "basic_auth" {
			req.SetBasicAuth(secret.Username, secret.Password)
			// log.Debug("Using basic_auth authentication: %s:%s", secret.Username, secret.Password)
		} else if auth.Type == "header" {
			// log.Debug("Using header authentication: %s = %s", auth.ParamName, secret.Key)
			req.Header.Add(auth.ParamName, secret.Key)
		}
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(tmpfile, resp.Body); err != nil {
			return "", err
		}
	} else if u.Scheme == "ftp" {
		// Download FTP
		p := u.Port()
		if p == "" {
			p = "21"
		}
		c, err := ftp.Dial(fmt.Sprintf("%s:%s", u.Hostname(), p), ftp.DialWithTimeout(10*time.Second))
		if err != nil {
			return "", err
		}
		if auth.Type != "basic_auth" {
			secret.Username = "anonymous"
			secret.Password = "anonymous"
		}
		err = c.Login(secret.Username, secret.Password)
		if err != nil {
			return "", err
		}
		r, err := c.Retr(u.Path)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(tmpfile, r); err != nil {
			return "", err
		}
	}
	return tmpfilepath, nil
}
