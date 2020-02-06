package dmfr

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/internal/log"
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
		log.Debug("Using query_param authentication: %s = %s", auth.ParamName, secret.Key)
	} else if auth.Type == "path_segment" {
		p := u.Path
		u.Path = strings.ReplaceAll(u.Path, "{}", secret.Key)
		log.Debug("Using path_segment authentication: %s -> %s", p, u.Path)
	}
	ustr := u.String()
	// Get the temporary file and full path
	tmpfile, err := ioutil.TempFile("", "fetch")
	if err != nil {
		return "", err
	}
	defer tmpfile.Close()
	tmpfilepath := tmpfile.Name()
	// Download
	client := &http.Client{}
	req, err := http.NewRequest("GET", ustr, nil)
	if err != nil {
		return "", err
	}
	if auth.Type == "basic_auth" {
		req.SetBasicAuth(secret.Username, secret.Password)
		log.Debug("Using basic_auth authentication: %s:%s", secret.Username, secret.Password)
	} else if auth.Type == "header" {
		log.Debug("Using header authentication: %s = %s", auth.ParamName, secret.Key)
		req.Header.Add(auth.ParamName, secret.Key)
	}
	log.Debug("AuthorizedRequest downloading %s -> %s", address, tmpfilepath)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	// Write the body to file
	_, err = io.Copy(tmpfile, resp.Body)
	return tmpfilepath, nil
}
