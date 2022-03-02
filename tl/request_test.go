package tl

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
)

func TestAuthorizedRequest(t *testing.T) {
	secret := Secret{Key: "abcd", Username: "efgh", Password: "ijkl"}
	testcases := []struct {
		name       string
		url        string
		auth       FeedAuthorization
		checkkey   string
		checkvalue string
	}{
		{
			"query_param",
			"http://httpbin.org/get",
			FeedAuthorization{Type: "query_param", ParamName: "api_key"},
			"url",
			"http://httpbin.org/get?api_key=abcd",
		},
		{
			"path_segment",
			"http://httpbin.org/anything/{}/ok",
			FeedAuthorization{Type: "path_segment"},
			"url",
			"http://httpbin.org/anything/abcd/ok",
		},
		{
			"header",
			"http://httpbin.org/headers",
			FeedAuthorization{Type: "header", ParamName: "Auth"},
			"", // TODO: check headers...
			"",
		},
		{
			"basic_auth",
			"http://httpbin.org/basic-auth/efgh/ijkl",
			FeedAuthorization{Type: "basic_auth"},
			"user",
			secret.Username,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tmpfile, err := AuthenticatedRequest(tc.url, secret, tc.auth)
			if err != nil {
				t.Error(err)
				return
			}
			p, err := ioutil.ReadFile(tmpfile)
			if err != nil {
				t.Error(err)
				return
			}
			var result map[string]interface{}
			if err := json.Unmarshal(p, &result); err != nil {
				t.Error(err)
				return
			}
			if tc.checkkey != "" {
				a, ok := result[tc.checkkey].(string)
				if !ok {
					t.Errorf("could not read key %s from response", tc.checkkey)
				} else if tc.checkvalue != a {
					t.Errorf("got %s, expected %s for response key %s", a, tc.checkvalue, tc.checkkey)
				}
			}
			if err := os.Remove(tmpfile); err != nil {
				t.Error(err)
			}
		})
	}
}
