package request

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/stretchr/testify/assert"
)

func TestAuthorizedRequest(t *testing.T) {
	// Any changes to test server will require adjusting size and sha1 in test cases below
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jb := make(map[string]interface{})
		jb["method"] = r.Method
		jb["url"] = r.URL.String()
		if a, b, ok := r.BasicAuth(); ok {
			jb["user"] = a
			jb["password"] = b
		}
		a, err := json.Marshal(jb)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.Header().Add("Status-Code", "200")
		w.Write(a)
	}))
	defer ts.Close()
	secret := tl.Secret{Key: "abcd", Username: "efgh", Password: "ijkl"}
	testcases := []struct {
		name       string
		url        string
		auth       tl.FeedAuthorization
		checkkey   string
		checkvalue string
		checksize  int
		checkcode  int
		checksha1  string
	}{
		{
			"basic get",
			"/get",
			tl.FeedAuthorization{},
			"url",
			"/get",
			29,
			200,
			"66621b979e91314ea163d94be8e7486bdcfe07c9",
		},
		{
			"query_param",
			"/get",
			tl.FeedAuthorization{Type: "query_param", ParamName: "api_key"},
			"url",
			"/get?api_key=abcd",
			42,
			200,
			"",
		},
		{
			"path_segment",
			"/anything/{}/ok",
			tl.FeedAuthorization{Type: "path_segment"},
			"url",
			"/anything/abcd/ok",
			0,
			200,
			"",
		},
		{
			"header",
			"/headers",
			tl.FeedAuthorization{Type: "header", ParamName: "Auth"},
			"", // TODO: check headers...
			"",
			0,
			200,
			"",
		},
		{
			"basic_auth",
			"/basic-auth/efgh/ijkl",
			tl.FeedAuthorization{Type: "basic_auth"},
			"user",
			secret.Username,
			0,
			200,
			"",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			d, responseSha1, responseSize, responseCode, err := AuthenticatedRequest(ts.URL+tc.url, WithAuth(secret, tc.auth))
			if err != nil {
				t.Error(err)
				return
			}
			var result map[string]interface{}
			if err := json.Unmarshal(d, &result); err != nil {
				t.Error(err)
				return
			}
			if tc.checksize > 0 {
				assert.Equal(t, tc.checksize, responseSize, "did not match expected size")
			}
			if tc.checkcode > 0 {
				assert.Equal(t, tc.checkcode, responseCode, "did not match expected response code")
			}
			if tc.checksha1 != "" {
				assert.Equal(t, tc.checksha1, responseSha1, "did not match expected sha1")
			}
			if tc.checkkey != "" {
				a, ok := result[tc.checkkey].(string)
				if !ok {
					t.Errorf("could not read key %s from response", tc.checkkey)
				} else if tc.checkvalue != a {
					t.Errorf("got %s, expected %s for response key %s", a, tc.checkvalue, tc.checkkey)
				}
			}
		})
	}
}
