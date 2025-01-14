package request

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/testpath"
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

func testBucket(t *testing.T, ctx context.Context, bucket Bucket) {
	testDir := "testdata/request"
	testRelkey := "testdata/request/readme.md"
	uploadKey := "ok.md"
	testFullkey := testpath.RelPath(testRelkey)
	checkfunc := func(b string) bool {
		return strings.HasSuffix(b, ".txt")
	}
	checkRtFiles, err := findFiles(testpath.RelPath(testDir), checkfunc)
	if err != nil {
		t.Fatal(err)
	}
	srcDir := testpath.RelPath(testDir)
	srcDirPrefix := "test-upload-all"

	////////
	localCheckDir, err := os.MkdirTemp("", "testBucketDownload")
	if err != nil {
		t.Fatal(err)
	}
	// defer os.RemoveAll(localCheckDir)
	///////
	t.Run("Upload", func(t *testing.T) {
		inf, err := os.Open(testFullkey)
		if err != nil {
			t.Fatal(err)
		}
		if err := bucket.Upload(ctx, uploadKey, inf); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Download", func(t *testing.T) {
		// Now check
		rio, _, err := bucket.Download(ctx, uploadKey)
		if err != nil {
			t.Fatal(err)
		}
		checkfn := filepath.Join(localCheckDir, "test.pb")
		if err := copyToFile(ctx, rio, checkfn); err != nil {
			t.Fatal(err)
		}
		if checkf, err := filesEqual(testFullkey, checkfn); err != nil {
			t.Fatal(err)
		} else if !checkf {
			t.Error("expected files to be equal")
		}
	})
	t.Run("UploadAll", func(t *testing.T) {
		if err := bucket.UploadAll(ctx, srcDir, srcDirPrefix, checkfunc); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("DownloadAll", func(t *testing.T) {
		downloadDir := filepath.Join(localCheckDir, "downloadAll")
		fns, err := bucket.DownloadAll(ctx, downloadDir, srcDirPrefix, checkfunc)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, len(checkRtFiles), len(fns), "expected number of downloaded files")
		for _, checkRtFn := range checkRtFiles {
			checkDownloadFn := filepath.Join(
				downloadDir,
				stripDir(testpath.RelPath("testdata/request"), checkRtFn),
			)
			if checkRelKey, err := filesEqual(checkRtFn, checkDownloadFn); err != nil {
				t.Fatal(err)
			} else if !checkRelKey {
				t.Error("expeced files to be equal")
			}

		}
	})
	if bucketSign, ok := bucket.(Presigner); ok {
		t.Run("CreateSignedUrl", func(t *testing.T) {
			// Upload file
			signKey := "test-upload.zip"
			testData := []byte("test file upload")
			data := bytes.NewBuffer(testData)
			if err := bucket.Upload(ctx, signKey, data); err != nil {
				t.Fatal(err)
			}
			// Download again
			signedUrl, err := bucketSign.CreateSignedUrl(ctx, signKey, "download.zip")
			if err != nil {
				t.Fatal(err)
			}
			resp, err := http.Get(signedUrl)
			if err != nil {
				t.Error(err)
			}
			downloadData, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if string(downloadData) != string(testData) {
				t.Errorf("got data '%s', expected '%s'", string(downloadData), string(testData))
			}
		})
	}
}

func filesEqual(a string, b string) (bool, error) {
	adata, err := os.ReadFile(a)
	if err != nil {
		return false, err
	}
	bdata, err := os.ReadFile(b)
	if err != nil {
		return false, err
	}
	return slices.Equal(adata, bdata), nil
}
