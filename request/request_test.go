package request

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/stretchr/testify/assert"
)

// nonSeekableReader wraps an io.Reader to make it non-seekable
type nonSeekableReader struct {
	r io.Reader
}

func (n *nonSeekableReader) Read(p []byte) (int, error) {
	return n.r.Read(p)
}

func TestMd5FromReader(t *testing.T) {
	testCases := []struct {
		name          string
		data          string
		seekable      bool
		expectedMD5   string // hex encoded, empty means nil expected
		checkPosReset bool
	}{
		{
			name:          "seekable reader with data",
			data:          "hello world",
			seekable:      true,
			expectedMD5:   "5eb63bbbe01eeed093cb22bb8f5acdc3",
			checkPosReset: true,
		},
		{
			name:          "seekable reader empty",
			data:          "",
			seekable:      true,
			expectedMD5:   "d41d8cd98f00b204e9800998ecf8427e", // MD5 of empty string
			checkPosReset: true,
		},
		{
			name:        "non-seekable reader returns nil",
			data:        "hello world",
			seekable:    false,
			expectedMD5: "",
		},
		{
			name:          "seekable reader with binary data",
			data:          "\x00\x01\x02\x03\x04\x05",
			seekable:      true,
			expectedMD5:   "d15ae53931880fd7b724dd7888b4b4ed",
			checkPosReset: true,
		},
		{
			name:          "seekable reader with unicode",
			data:          "héllo wörld 🌍",
			seekable:      true,
			expectedMD5:   "0e487ea323b9fe6a06a7796b48540d5c",
			checkPosReset: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var reader io.Reader
			var seeker io.ReadSeeker

			if tc.seekable {
				seeker = strings.NewReader(tc.data)
				reader = seeker
			} else {
				reader = &nonSeekableReader{r: strings.NewReader(tc.data)}
			}

			result := md5FromReader(reader)

			if tc.expectedMD5 == "" {
				assert.Nil(t, result, "expected nil for non-seekable reader")
			} else {
				assert.NotNil(t, result, "expected non-nil MD5 hash")
				gotMD5 := fmt.Sprintf("%x", result)
				assert.Equal(t, tc.expectedMD5, gotMD5, "MD5 hash mismatch")
			}

			// Verify reader position is reset to start for seekable readers
			if tc.checkPosReset && tc.seekable {
				// Read first byte to verify position
				buf := make([]byte, 1)
				n, err := reader.Read(buf)
				if len(tc.data) > 0 {
					assert.NoError(t, err, "should be able to read after MD5 calculation")
					assert.Equal(t, 1, n, "should read 1 byte")
					assert.Equal(t, tc.data[0], buf[0], "first byte should match original data")
				}
			}
		})
	}
}

func TestMd5FromReader_PositionPreserved(t *testing.T) {
	// Test that reader position is properly reset even when starting mid-stream
	data := "abcdefghij"
	reader := strings.NewReader(data)

	// Read some bytes first to move position
	buf := make([]byte, 3)
	_, err := reader.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, "abc", string(buf))

	// Calculate MD5 - should calculate for entire content
	result := md5FromReader(reader)
	assert.NotNil(t, result)

	expectedMD5 := "a925576942e94b2ef57a066101b48876" // MD5 of "abcdefghij"
	gotMD5 := fmt.Sprintf("%x", result)
	assert.Equal(t, expectedMD5, gotMD5)

	// Reader should be back at start
	fullBuf := make([]byte, 10)
	n, err := reader.Read(fullBuf)
	assert.NoError(t, err)
	assert.Equal(t, 10, n)
	assert.Equal(t, data, string(fullBuf))
}

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
		auth        dmfr.FeedAuthorization
		checkkey    string
		checkvalue  string
		checksize   int
		checkcode   int
		checksha1   string
		expectError bool
		secret      dmfr.Secret
	}{
		{
			name:       "basic get",
			url:        "/get",
			auth:       dmfr.FeedAuthorization{},
			checkkey:   "url",
			checkvalue: "/get",
			checksize:  29,
			checkcode:  200,
			checksha1:  "66621b979e91314ea163d94be8e7486bdcfe07c9",
		},
		{
			name:       "query_param",
			url:        "/get",
			auth:       dmfr.FeedAuthorization{Type: "query_param", ParamName: "api_key"},
			checkkey:   "url",
			checkvalue: "/get?api_key=abcd",
			checksize:  42,
			checkcode:  200,
			checksha1:  "",
			secret:     dmfr.Secret{Key: "abcd"},
		},
		{
			name:       "path_segment",
			url:        "/anything/{}/ok",
			auth:       dmfr.FeedAuthorization{Type: "path_segment"},
			checkkey:   "url",
			checkvalue: "/anything/abcd/ok",
			checksize:  0,
			checkcode:  200,
			checksha1:  "",
			secret:     dmfr.Secret{Key: "abcd"},
		},
		{
			name:       "header",
			url:        "/headers",
			auth:       dmfr.FeedAuthorization{Type: "header", ParamName: "Auth"},
			checkkey:   "", // TODO: check headers...
			checkvalue: "",
			checksize:  0,
			checkcode:  200,
			checksha1:  "",
			secret:     dmfr.Secret{Key: "abcd"},
		},
		{
			name:       "basic_auth",
			url:        "/basic-auth/efgh/ijkl",
			auth:       dmfr.FeedAuthorization{Type: "basic_auth"},
			checkkey:   "user",
			checkvalue: "efgh",
			checksize:  0,
			checkcode:  200,
			checksha1:  "",
			secret:     dmfr.Secret{Username: "efgh", Password: "ijkl"},
		},
		{
			name:       "replace",
			url:        "/get",
			auth:       dmfr.FeedAuthorization{Type: "replace_url"},
			checkkey:   "url",
			checkvalue: "/anything/test",
			checksize:  0,
			checkcode:  200,
			checksha1:  "",
			secret:     dmfr.Secret{ReplaceUrl: ts.URL + "/anything/test"},
		},
		{
			name:        "replace expect error",
			url:         "/get",
			auth:        dmfr.FeedAuthorization{Type: "replace_url"},
			expectError: true,
			secret:      dmfr.Secret{ReplaceUrl: "/must/be/full/url"},
		},
	}
	ctx := context.TODO()
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			fr, err := AuthenticatedRequest(ctx, &out, ts.URL+tc.url, WithAuth(tc.secret, tc.auth))
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
			if err := json.Unmarshal(out.Bytes(), &result); err != nil {
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
