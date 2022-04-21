package fetch

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/stretchr/testify/assert"
)

func TestStaticFetch(t *testing.T) {
	basedir := ""
	tcs := []struct {
		name          string
		serveFile     string
		requestPath   string
		responseSha1  string
		responseCode  int
		responseError bool
		fvFound       bool
		fvSha1        string
	}{
		{
			"example.zip",
			"test/data/example.zip",
			"test/data/example.zip",
			"ce0a38dd6d4cfdac6aebe003181b6b915390a3b8",
			200,
			false,
			false,
			"ce0a38dd6d4cfdac6aebe003181b6b915390a3b8",
		},
		{
			"404",
			"test/data/example.zip",
			"404.zip",
			"",
			404,
			true,
			false,
			"",
		},
		{
			"invalid zip",
			"test/data/invalid.zip",
			"test/data/invalid.zip",
			"",
			200,
			true,
			false,
			"",
		},
		{
			"nested dir",
			"test/data/example-nested-dir.zip",
			"test/data/example-nested-dir.zip#example-nested-dir/example",
			"",
			200,
			false,
			false,
			"97ae78529b47860f3d5b674f27121c078f7b3402",
		},
		{
			"nested zip",
			"test/data/example-nested-zip.zip",
			"test/data/example-nested-zip.zip#example-nested-zip/example.zip",
			"",
			200,
			false,
			false,
			"ce0a38dd6d4cfdac6aebe003181b6b915390a3b8",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/"+tc.serveFile {
					http.Error(w, "404", 404)
					return
				}
				buf, err := ioutil.ReadFile(testutil.RelPath(basedir + "/" + tc.serveFile))
				if err != nil {
					t.Error(err)
				}
				w.Write(buf)
			}))
			defer ts.Close()
			tmpdir, err := ioutil.TempDir("", "gtfs")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpdir) // clean up
			testdb.WithAdapterRollback(func(atx tldb.Adapter) error {
				url := ts.URL + "/" + tc.requestPath
				feed := testdb.CreateTestFeed(atx, url)
				fv, fr, err := StaticFetch(atx, feed, Options{FeedURL: url, Directory: tmpdir})
				if err != nil {
					t.Error(err)
					return err
				}
				assert.Equal(t, tc.fvFound, fr.Found, "did not get expected found value")
				assert.Equal(t, tc.responseCode, fr.ResponseCode, "did not get expected response code")
				assert.Equal(t, tc.responseError, fr.FetchError != nil, "did not get expected value for fetch error")
				if tc.responseError {
					if fr.FetchError == nil {
						t.Errorf("expected fetch error, got none")
					}
				} else if fr.FetchError != nil {
					t.Errorf("got unexpected error: %s", fr.FetchError.Error())
				}
				if tc.responseSha1 != "" {
					assert.Equal(t, tc.responseSha1, fr.ResponseSHA1, "did not get expected response sha1")
				}
				//
				tlff := dmfr.FeedFetch{}
				testdb.ShouldGet(t, atx, &tlff, `SELECT * FROM feed_fetches WHERE feed_id = ? ORDER BY id DESC LIMIT 1`, feed.ID)
				assert.Equal(t, tc.responseCode, tlff.ResponseCode.Int, "did not get expected feed_fetch response code")
				assert.Equal(t, !tc.responseError, tlff.Success, "did not get expected feed_fetch success")
				//
				if !tc.responseError {
					fv2 := tl.FeedVersion{ID: fv.ID}
					testdb.ShouldFind(t, atx, &fv2)
					assert.Equal(t, url, fv2.URL, "did not get expected feed version url")
					assert.Equal(t, tc.fvSha1, fv.SHA1, "did not get expected feed version sha1")
					assert.Equal(t, feed.ID, fv2.FeedID, "did not get expected feed version feed ID")
				}
				return nil
			})
		})
	}
}

func TestStaticFetch_Exists(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile(ExampleZip.URL)
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))
	testdb.WithAdapterRollback(func(atx tldb.Adapter) error {
		url := ts.URL
		feed := testdb.CreateTestFeed(atx, url)
		fv, _, err := StaticFetch(atx, feed, Options{FeedURL: url, Directory: ""})
		if err != nil {
			t.Fatal(err)
		}
		fv2, fr2, err2 := StaticFetch(atx, feed, Options{FeedURL: url, Directory: ""})
		if err2 != nil {
			t.Error(err2)
		}
		if !(fr2.Found) {
			t.Error("expected found feed")
		}
		if fv2.SHA1 != ExampleZip.SHA1 {
			t.Errorf("got %s expect %s", fv2.SHA1, ExampleZip.SHA1)
		}
		if fv2.ID == 0 {
			t.Error("expected non-zero value")
		}
		if fv.ID != fv2.ID {
			t.Errorf("got %d expected %d", fv.ID, fv2.ID)
		}
		return nil
	})
}
