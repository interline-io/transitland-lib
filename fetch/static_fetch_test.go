package fetch

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/stretchr/testify/assert"
)

var ExampleZip = testutil.ExampleZip

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
			name:          "example.zip",
			serveFile:     "testdata/example.zip",
			requestPath:   "testdata/example.zip",
			responseSha1:  "ce0a38dd6d4cfdac6aebe003181b6b915390a3b8",
			responseCode:  200,
			responseError: false,
			fvFound:       false,
			fvSha1:        "ce0a38dd6d4cfdac6aebe003181b6b915390a3b8",
		},
		{
			name:          "404",
			serveFile:     "testdata/example.zip",
			requestPath:   "404.zip",
			responseSha1:  "",
			responseCode:  404,
			responseError: true,
			fvFound:       false,
			fvSha1:        "",
		},
		{
			name:          "invalid zip",
			serveFile:     "testdata/invalid.zip",
			requestPath:   "testdata/invalid.zip",
			responseSha1:  "",
			responseCode:  200,
			responseError: true,
			fvFound:       false,
			fvSha1:        "",
		},
		{
			name:          "nested dir",
			serveFile:     "testdata/example-nested-dir.zip",
			requestPath:   "testdata/example-nested-dir.zip#example-nested-dir/example",
			responseSha1:  "",
			responseCode:  200,
			responseError: false,
			fvFound:       false,
			fvSha1:        "97ae78529b47860f3d5b674f27121c078f7b3402",
		},
		{
			name:          "nested two feeds 1",
			serveFile:     "testdata/example-nested-two-feeds.zip",
			requestPath:   "testdata/example-nested-two-feeds.zip#example1",
			responseSha1:  "",
			responseCode:  200,
			responseError: false,
			fvFound:       false,
			fvSha1:        "196bc2b5ff85d629e279e3fbfc9e05c520075fba",
		},
		{
			name:          "nested two feeds 2",
			serveFile:     "testdata/example-nested-two-feeds.zip",
			requestPath:   "testdata/example-nested-two-feeds.zip#example2",
			responseSha1:  "",
			responseCode:  200,
			responseError: false,
			fvFound:       false,
			fvSha1:        "196bc2b5ff85d629e279e3fbfc9e05c520075fba",
		},
		{
			name:          "nested zip",
			serveFile:     "testdata/example-nested-zip.zip",
			requestPath:   "testdata/example-nested-zip.zip#example-nested-zip/example.zip",
			responseSha1:  "",
			responseCode:  200,
			responseError: false,
			fvFound:       false,
			fvSha1:        "ce0a38dd6d4cfdac6aebe003181b6b915390a3b8",
		},
	}
	ctx := context.TODO()
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/"+tc.serveFile {
					http.Error(w, "404", 404)
					return
				}
				buf, err := ioutil.ReadFile(testpath.RelPath(basedir + "/" + tc.serveFile))
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
				fr, err := StaticFetch(ctx, atx, Options{FeedID: feed.ID, FeedURL: url, Storage: tmpdir})
				if err != nil {
					t.Error(err)
					return err
				}
				fv := fr.FeedVersion
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
				assert.Equal(t, tc.responseCode, tlff.ResponseCode.Int(), "did not get expected feed_fetch response code")
				assert.Equal(t, !tc.responseError, tlff.Success, "did not get expected feed_fetch success")
				//
				if !tc.responseError {
					fv2 := dmfr.FeedVersion{}
					fv2.ID = fv.ID
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
	ctx := context.TODO()
	testdb.TempSqlite(func(atx tldb.Adapter) error {
		url := ts.URL
		feed := testdb.CreateTestFeed(atx, url)
		_ = feed
		tmpdir := t.TempDir()
		fr1, err := StaticFetch(ctx, atx, Options{FeedID: feed.ID, FeedURL: url, Storage: tmpdir})
		if err != nil {
			t.Fatal(err)
		}
		fr2, err2 := StaticFetch(ctx, atx, Options{FeedID: feed.ID, FeedURL: url, Storage: tmpdir})
		if err2 != nil {
			t.Error(err2)
		}
		if !(fr2.Found) {
			t.Error("expected found feed")
		}
		fv1 := fr1.FeedVersion
		fv2 := fr2.FeedVersion
		if fv2.SHA1 != ExampleZip.SHA1 {
			t.Errorf("got %s expect %s", fv2.SHA1, ExampleZip.SHA1)
		}
		if fv2.ID == 0 {
			t.Error("expected non-zero value")
		}
		if fv1.ID != fv2.ID {
			t.Errorf("got %d expected %d", fv1.ID, fv2.ID)
		}
		if fr2.FeedVersionID.Int() != fv1.ID {
			t.Errorf("got %d expected %d as feed version id in result", fr2.FeedVersionID.Int(), fv1.ID)
		}
		return nil
	})
}

func TestStaticFetch_AdditionalTests(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile(ExampleZip.URL)
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))
	defer ts.Close()
	ctx := context.TODO()
	testdb.TempSqlite(func(atx tldb.Adapter) error {
		tmpdir, err := ioutil.TempDir("", "gtfs")
		if err != nil {
			t.Error(err)
			return nil
		}
		defer os.RemoveAll(tmpdir) // clean up
		//
		url := ts.URL
		feed := testdb.CreateTestFeed(atx, ts.URL)
		fr, err := StaticFetch(ctx, atx, Options{FeedID: feed.ID, FeedURL: feed.URLs.StaticCurrent, Storage: tmpdir})
		if err != nil {
			t.Error(err)
			return nil
		}
		if fr.FetchError != nil {
			t.Error(fr.FetchError)
			return nil
		}
		if fr.Found {
			t.Errorf("expected new fv")
			return nil
		}

		// Check FV
		fv := fr.FeedVersion
		if fv.SHA1 != ExampleZip.SHA1 {
			t.Errorf("got %s expect %s", fv.SHA1, ExampleZip.SHA1)
			return nil
		}
		fv2 := dmfr.FeedVersion{}
		fv2.ID = fv.ID
		testdb.ShouldFind(t, atx, &fv2)
		if fv2.URL != url {
			t.Errorf("got %s expect %s", fv2.URL, url)
		}
		if fv2.FeedID != feed.ID {
			t.Errorf("got %d expect %d", fv2.FeedID, feed.ID)
		}
		if fv2.SHA1 != ExampleZip.SHA1 {
			t.Errorf("got %s expect %s", fv2.SHA1, ExampleZip.SHA1)
		}

		// Check FeedFetch record
		tlff := dmfr.FeedFetch{}
		testdb.ShouldGet(t, atx, &tlff, `SELECT * FROM feed_fetches WHERE feed_id = ? ORDER BY id DESC LIMIT 1`, feed.ID)
		assert.Equal(t, fv.SHA1, tlff.ResponseSHA1.Val, "did not get expected feed_fetch sha1")
		assert.Equal(t, 200, tlff.ResponseCode.Int(), "did not get expected feed_fetch response code")
		assert.Equal(t, true, tlff.Success, "did not get expected feed_fetch success")

		// Check that we saved the output file
		outfn := filepath.Join(tmpdir, fv.SHA1+".zip")
		info, err := os.Stat(outfn)
		if os.IsNotExist(err) {
			t.Fatalf("expected file to exist: %s", outfn)
		}
		expsize := int64(ExampleZip.Size)
		if info.Size() != expsize {
			t.Errorf("got %d bytes in file, expected %d", info.Size(), expsize)
		}
		return nil
	})
}

// Currently we cannot support two "directory" type feeds nested inside the same zip
// Fetch returns the sha1 of the "whole" zip file unless a nested .zip is extracted
// So in this case, the second fetch will return Found and the existing FV.
func TestStaticFetch_NestedTwoFeeds(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fp := testpath.RelPath("testdata/example-nested-two-feeds.zip")
		buf, err := ioutil.ReadFile(fp)
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))
	defer ts.Close()
	ctx := context.TODO()
	testdb.TempSqlite(func(atx tldb.Adapter) error {
		tmpdir, err := ioutil.TempDir("", "gtfs")
		if err != nil {
			t.Error(err)
			return nil
		}
		defer os.RemoveAll(tmpdir) // clean up
		//
		tcs := []struct {
			url   string
			found bool
		}{
			{url: "test.zip#example1", found: false},
			{url: "test.zip#example2", found: true},
		}
		for _, tc := range tcs {
			_ = tc
			feed := testdb.CreateTestFeed(atx, ts.URL+"/"+tc.url)
			fr, err := StaticFetch(ctx, atx, Options{FeedID: feed.ID, FeedURL: feed.URLs.StaticCurrent, Storage: tmpdir})
			if err != nil {
				t.Error(err)
				return nil
			}
			if fr.FetchError != nil {
				t.Error(fr.FetchError)
				return nil
			}
			if fr.Found != tc.found {
				t.Errorf("expected found to be %t, got %t", tc.found, fr.Found)
			}
		}
		return nil
	})
}

// func TestStaticFetch_CreateFeed(t *testing.T) {
// 	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		buf, err := ioutil.ReadFile(ExampleZip.URL)
// 		if err != nil {
// 			t.Error(err)
// 		}
// 		w.Write(buf)
// 	}))
// 	defer ts.Close()
// 	testdb.TempSqlite(func(atx tldb.Adapter) error {
// 		tmpdir, err := ioutil.TempDir("", "gtfs")
// 		if err != nil {
// 			t.Error(err)
// 			return nil
// 		}
// 		defer os.RemoveAll(tmpdir) // clean up
// 		//
// 		url := ts.URL
// 		feed := dmfr.Feed{}
// 		feed.FeedID = "caltrain"
// 		fv, _, err := StaticFetch(atx, Options{FeedID: feed.ID, FeedURL: ts.URL, FeedCreate: true, Directory: tmpdir})
// 		if err != nil {
// 			t.Error(err)
// 			return nil
// 		}
// 		// Check Feed
// 		tf2 := dmfr.Feed{}
// 		testdb.ShouldGet(t, atx, &tf2, `SELECT * FROM current_feeds WHERE onestop_id = ?`, "caltrain")
// 		// Check FV
// 		fv2 := dmfr.FeedVersion{ID: fv.ID}
// 		testdb.ShouldFind(t, atx, &fv2)
// 		if fv2.URL != url {
// 			t.Errorf("got %s expect %s", fv2.URL, url)
// 		}
// 		if fv2.SHA1 != ExampleZip.SHA1 {
// 			t.Errorf("got %s expect %s", fv2.SHA1, ExampleZip.SHA1)
// 		}
// 		return nil
// 	})
// }

func TestStaticStateFetch_FetchError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", 404)
	}))
	defer ts.Close()
	ctx := context.TODO()
	testdb.TempSqlite(func(atx tldb.Adapter) error {
		tmpdir, err := ioutil.TempDir("", "gtfs")
		if err != nil {
			t.Error(err)
			return nil
		}
		defer os.RemoveAll(tmpdir) // clean up
		feed := testdb.CreateTestFeed(atx, ts.URL)
		// Fetch
		_, err = StaticFetch(ctx, atx, Options{FeedID: feed.ID, FeedURL: feed.URLs.StaticCurrent, Storage: tmpdir})
		if err != nil {
			t.Error(err)
			return nil
		}
		// Check FeedFetch record
		tlff := dmfr.FeedFetch{}
		testdb.ShouldGet(t, atx, &tlff, `SELECT * FROM feed_fetches WHERE feed_id = ? ORDER BY id DESC LIMIT 1`, feed.ID)
		assert.Equal(t, 404, tlff.ResponseCode.Int(), "did not get expected feed_fetch response code")
		assert.Equal(t, false, tlff.Success, "did not get expected feed_fetch success")
		return nil
	})
}

func TestStaticStateFetch_HideURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile(ExampleZip.URL)
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))
	defer ts.Close()
	ctx := context.TODO()
	testdb.TempSqlite(func(atx tldb.Adapter) error {
		tmpdir, err := ioutil.TempDir("", "gtfs")
		if err != nil {
			t.Error(err)
			return nil
		}
		defer os.RemoveAll(tmpdir) // clean up
		feed := testdb.CreateTestFeed(atx, ts.URL)
		// Fetch
		_, err = StaticFetch(ctx, atx, Options{FeedID: feed.ID, FeedURL: feed.URLs.StaticCurrent, Storage: tmpdir, HideURL: true})
		if err != nil {
			t.Error(err)
			return nil
		}
		// Check FeedFetch record
		tlff := dmfr.FeedFetch{}
		testdb.ShouldGet(t, atx, &tlff, `SELECT * FROM feed_fetches WHERE feed_id = ? ORDER BY id DESC LIMIT 1`, feed.ID)
		assert.Equal(t, 200, tlff.ResponseCode.Int(), "did not get expected feed_fetch response code")
		assert.Equal(t, true, tlff.Success, "did not get expected feed_fetch success")
		assert.Equal(t, "", tlff.URL, "feed fetch url")
		return nil
	})
}
