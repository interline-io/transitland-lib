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
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/stretchr/testify/assert"
)

func TestRTFetch(t *testing.T) {
	basedir := "test/data/rt"
	tcs := []struct {
		name          string
		serveFile     string
		requestPath   string
		fvFound       bool
		responseSha1  string
		responseCode  int
		responseError bool
		entityCount   int
	}{
		{"example.pb", "example.pb", "example.pb", false, "1cb30340f47b5ced4238c8085f0d5bb1dffd6207", 200, false, 26},
		{"404", "example.pb", "404.pb", false, "", 404, true, 0},
		{"invalid", "invalid.pb", "invalid.pb", false, "cc0fcdb9351ee7cf357afc548236eff75acd8327", 200, true, 0},
	}
	for _, tc := range tcs {
		t.Run("", func(t *testing.T) {
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
			testdb.TempSqlite(func(atx tldb.Adapter) error {
				url := ts.URL + "/" + tc.requestPath
				feed := testdb.CreateTestFeed(atx, url)
				msg, fr, err := RTFetch(atx, Options{FeedID: feed.ID, FeedURL: url, Directory: tmpdir})
				if err != nil {
					t.Error(err)
					return err
				}
				if msg != nil {
					assert.Equal(t, tc.entityCount, len(msg.Entity), "did not get expected rt feed entity count")
				}
				assert.Equal(t, tc.fvFound, fr.Found, "did not get expected found value")
				if tc.responseSha1 != "" {
					assert.Equal(t, tc.responseSha1, fr.ResponseSHA1, "did not get expected response sha1")
				}
				if tc.responseCode > 0 {
					assert.Equal(t, tc.responseCode, fr.ResponseCode, "did not get expected response code")
				}
				assert.Equal(t, tc.responseError, fr.FetchError != nil)
				//
				tlff := dmfr.FeedFetch{}
				testdb.ShouldGet(t, atx, &tlff, `SELECT * FROM feed_fetches WHERE feed_id = ? ORDER BY id DESC LIMIT 1`, feed.ID)
				assert.Equal(t, tc.responseCode, tlff.ResponseCode.Int, "did not get expected feed_fetch response code")
				assert.Equal(t, !tc.responseError, tlff.Success, "did not get expected feed_fetch success")
				return nil
			})
		})
	}
}
