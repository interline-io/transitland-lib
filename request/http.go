package request

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	tl "github.com/interline-io/transitland-lib"
	"github.com/interline-io/transitland-lib/dmfr"
)

type Http struct{}

func (r Http) Download(ctx context.Context, ustr string, secret dmfr.Secret, auth dmfr.FeedAuthorization) (io.ReadCloser, int, error) {
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
	} else if auth.Type == "replace_url" {
		u, err = url.Parse(secret.ReplaceUrl)
		if err != nil {
			return nil, 0, errors.New("could not parse replacement query string")
		}
	}
	ustr = u.String()

	// Prepare HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", ustr, nil)
	if err != nil {
		return nil, 0, errors.New("invalid request")
	}

	// Set basic auth, if used
	if auth.Type == "basic_auth" {
		req.SetBasicAuth(secret.Username, secret.Password)
	} else if auth.Type == "header" {
		req.Header.Add(auth.ParamName, secret.Key)
	}

	// Make HTTP request
	req.Header.Set("User-Agent", fmt.Sprintf("transitland/%s", tl.Version.Tag))

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
