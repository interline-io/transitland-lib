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
		file        string
		reqpath     string
		found       bool
		sha1        string
		code        int
		expectError bool
	}{
		{"test/data/example.zip", "test/data/example.zip", false, "ce0a38dd6d4cfdac6aebe003181b6b915390a3b8", 200, false},
		{"test/data/example.zip", "404.zip", false, "", 404, true},
		{"test/data/invalid.zip", "test/data/invalid.zip", false, "", 200, true},
	}
	for _, tc := range tcs {
		t.Run("", func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/"+tc.file {
					http.Error(w, "404", 404)
					return
				}
				buf, err := ioutil.ReadFile(testutil.RelPath(basedir + "/" + tc.file))
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
				url := ts.URL + "/" + tc.reqpath
				feed := testdb.CreateTestFeed(atx, url)
				fr, err := StaticFetch(atx, feed, Options{FeedURL: url, Directory: tmpdir})
				if err != nil {
					t.Error(err)
					return err
				}
				assert.Equal(t, tc.found, fr.Found, "did not get expected found value")
				assert.Equal(t, tc.code, fr.ResponseCode, "did not get expected response code")
				assert.Equal(t, tc.expectError, fr.FetchError != nil)
				if tc.sha1 != "" {
					assert.Equal(t, tc.sha1, fr.ResponseSHA1, "did not get expected response sha1")
				}
				//
				tlff := dmfr.FeedFetch{}
				testdb.ShouldGet(t, atx, &tlff, `SELECT * FROM feed_fetches WHERE feed_id = ? ORDER BY id DESC LIMIT 1`, feed.ID)
				assert.Equal(t, tc.code, tlff.ResponseCode.Int, "did not get expected feed_fetch response code")
				assert.Equal(t, !tc.expectError, tlff.Success, "did not get expected feed_fetch success")
				//
				if !tc.expectError {
					assert.Equal(t, tc.sha1, fr.FeedVersion.SHA1)
					fv2 := tl.FeedVersion{ID: fr.FeedVersion.ID}
					testdb.ShouldFind(t, atx, &fv2)
					assert.Equal(t, url, fv2.URL)
					assert.Equal(t, feed.ID, fv2.FeedID)
					assert.Equal(t, tc.sha1, fv2.SHA1)
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
		fr, err := StaticFetch(atx, feed, Options{FeedURL: url, Directory: ""})
		if err != nil {
			t.Fatal(err)
		}
		fr2, err2 := StaticFetch(atx, feed, Options{FeedURL: url, Directory: ""})
		if err2 != nil {
			t.Error(err2)
		}
		if !(fr2.Found) {
			t.Error("expected found feed")
		}
		if fr2.FeedVersion.SHA1 != ExampleZip.SHA1 {
			t.Errorf("got %s expect %s", fr2.FeedVersion.SHA1, ExampleZip.SHA1)
		}
		if fr2.FeedVersion.ID == 0 {
			t.Error("expected non-zero value")
		}
		if fr.FeedVersion.ID != fr2.FeedVersion.ID {
			t.Errorf("got %d expected %d", fr.FeedVersion.ID, fr2.FeedVersion.ID)
		}
		return nil
	})
}
