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
	testcases := []struct {
		name        string
		url         string
		auth        tl.FeedAuthorization
		checkkey    string
		checkvalue  string
		checksize   int
		checkcode   int
		checksha1   string
		expectError bool
		secret      tl.Secret
	}{
		{
			name:       "basic get",
			url:        "/get",
			auth:       tl.FeedAuthorization{},
			checkkey:   "url",
			checkvalue: "/get",
			checksize:  29,
			checkcode:  200,
			checksha1:  "66621b979e91314ea163d94be8e7486bdcfe07c9",
		},
		{
			name:       "query_param",
			url:        "/get",
			auth:       tl.FeedAuthorization{Type: "query_param", ParamName: "api_key"},
			checkkey:   "url",
			checkvalue: "/get?api_key=abcd",
			checksize:  42,
			checkcode:  200,
			checksha1:  "",
			secret:     tl.Secret{Key: "abcd"},
		},
		{
			name:       "path_segment",
			url:        "/anything/{}/ok",
			auth:       tl.FeedAuthorization{Type: "path_segment"},
			checkkey:   "url",
			checkvalue: "/anything/abcd/ok",
			checksize:  0,
			checkcode:  200,
			checksha1:  "",
			secret:     tl.Secret{Key: "abcd"},
		},
		{
			name:       "header",
			url:        "/headers",
			auth:       tl.FeedAuthorization{Type: "header", ParamName: "Auth"},
			checkkey:   "", // TODO: check headers...
			checkvalue: "",
			checksize:  0,
			checkcode:  200,
			checksha1:  "",
			secret:     tl.Secret{Key: "abcd"},
		},
		{
			name:       "basic_auth",
			url:        "/basic-auth/efgh/ijkl",
			auth:       tl.FeedAuthorization{Type: "basic_auth"},
			checkkey:   "user",
			checkvalue: "efgh",
			checksize:  0,
			checkcode:  200,
			checksha1:  "",
			secret:     tl.Secret{Username: "efgh", Password: "ijkl"},
		},
		{
			name:       "replace",
			url:        "/get",
			auth:       tl.FeedAuthorization{Type: "replace"},
			checkkey:   "url",
			checkvalue: "/anything/test",
			checksize:  0,
			checkcode:  200,
			checksha1:  "",
			secret:     tl.Secret{Key: ts.URL + "/anything/test"},
		},
		{
			name:        "replace expect error",
			url:         "/get",
			auth:        tl.FeedAuthorization{Type: "replace"},
			expectError: true,
			secret:      tl.Secret{Key: "/must/be/full/url"},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			fr, err := AuthenticatedRequest(ts.URL+tc.url, WithAuth(tc.secret, tc.auth))
			if err != nil {
				t.Error(err)
				return
			}
			ferr := fr.FetchError
			if tc.expectError && ferr != nil {
				// ok
				return
			} else if tc.expectError && ferr == nil {
				t.Error("expected error")
			} else if !tc.expectError && ferr != nil {
				t.Error(ferr)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(fr.Data, &result); err != nil {
				t.Error(err)
				return
			}
			if tc.checksize > 0 {
				assert.Equal(t, tc.checksize, fr.ResponseSize, "did not match expected size")
			}
			if tc.checkcode > 0 {
				assert.Equal(t, tc.checkcode, fr.ResponseCode, "did not match expected response code")
			}
			if tc.checksha1 != "" {
				assert.Equal(t, tc.checksha1, fr.ResponseSHA1, "did not match expected sha1")
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
