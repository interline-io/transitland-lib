package fetch

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/stretchr/testify/assert"
)

var ExampleZip = testutil.ExampleZip

func TestFeedStateFetch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile(ExampleZip.URL)
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))
	defer ts.Close()
	testdb.WithAdapterRollback(func(atx tldb.Adapter) error {
		tmpdir, err := ioutil.TempDir("", "gtfs")
		if err != nil {
			t.Error(err)
			return nil
		}
		defer os.RemoveAll(tmpdir) // clean up
		//
		url := ts.URL
		feed := testdb.CreateTestFeed(atx, ts.URL)
		fr, err := FeedStateFetch(atx, Options{FeedID: feed.FeedID, Directory: tmpdir})
		if err != nil {
			t.Error(err)
			return nil
		}
		if fr.Found {
			t.Errorf("expected new fv")
			return nil
		}
		if fr.FeedVersion.SHA1 != ExampleZip.SHA1 {
			t.Errorf("got %s expect %s", fr.FeedVersion.SHA1, ExampleZip.SHA1)
			return nil
		}
		// Check FV
		fv2 := tl.FeedVersion{ID: fr.FeedVersion.ID}
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
		// Check FeedState
		tlf := dmfr.FeedState{}
		testdb.ShouldGet(t, atx, &tlf, `SELECT * FROM feed_states WHERE feed_id = ?`, feed.ID)
		if !tlf.LastSuccessfulFetchAt.Valid {
			t.Errorf("expected non-nil value")
		}
		// Check FeedFetch record
		tlff := dmfr.FeedFetch{}
		testdb.ShouldGet(t, atx, &tlff, `SELECT * FROM feed_fetches WHERE feed_id = ? ORDER BY id DESC LIMIT 1`, feed.ID)
		assert.Equal(t, fr.FeedVersion.SHA1, tlff.ResponseSHA1.String, "did not get expected feed_fetch sha1")
		assert.Equal(t, 200, tlff.ResponseCode.Int, "did not get expected feed_fetch response code")
		assert.Equal(t, true, tlff.Success, "did not get expected feed_fetch success")
		// Check that we saved the output file
		outfn := filepath.Join(tmpdir, fr.FeedVersion.SHA1+".zip")
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

func TestFeedStateFetch_CreateFeed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile(ExampleZip.URL)
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))
	defer ts.Close()
	testdb.WithAdapterRollback(func(atx tldb.Adapter) error {
		tmpdir, err := ioutil.TempDir("", "gtfs")
		if err != nil {
			t.Error(err)
			return nil
		}
		defer os.RemoveAll(tmpdir) // clean up
		//
		url := ts.URL
		fr, err := FeedStateFetch(atx, Options{FeedID: "caltrain", FeedURL: ts.URL, FeedCreate: true, Directory: tmpdir})
		if err != nil {
			t.Error(err)
			return nil
		}
		// Check Feed
		tf2 := tl.Feed{}
		testdb.ShouldGet(t, atx, &tf2, `SELECT * FROM current_feeds WHERE onestop_id = ?`, "caltrain")
		// Check FV
		fv2 := tl.FeedVersion{ID: fr.FeedVersion.ID}
		testdb.ShouldFind(t, atx, &fv2)
		if fv2.URL != url {
			t.Errorf("got %s expect %s", fv2.URL, url)
		}
		if fv2.SHA1 != ExampleZip.SHA1 {
			t.Errorf("got %s expect %s", fv2.SHA1, ExampleZip.SHA1)
		}
		// Check FeedState
		tlf := dmfr.FeedState{}
		testdb.ShouldGet(t, atx, &tlf, `SELECT * FROM feed_states WHERE feed_id = ?`, fv2.FeedID)
		if !tlf.LastSuccessfulFetchAt.Valid {
			t.Errorf("expected non-nil value")
		}
		return nil
	})
}

func TestFeedStateFetch_LastFetchError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", 404)
	}))
	defer ts.Close()
	testdb.WithAdapterRollback(func(atx tldb.Adapter) error {
		tmpdir, err := ioutil.TempDir("", "gtfs")
		if err != nil {
			t.Error(err)
			return nil
		}
		defer os.RemoveAll(tmpdir) // clean up
		feed := testdb.CreateTestFeed(atx, ts.URL)
		// Fetch
		_, err = FeedStateFetch(atx, Options{FeedID: feed.FeedID, Directory: tmpdir})
		if err != nil {
			t.Error(err)
			return nil
		}
		// Check FeedState
		tlf := dmfr.FeedState{}
		testdb.ShouldGet(t, atx, &tlf, `SELECT * FROM feed_states WHERE feed_id = ?`, feed.ID)
		experr := "response status code: 404"
		if tlf.LastFetchError == "" {
			t.Errorf("expected value for LastFetchError")
		}
		if !strings.HasPrefix(tlf.LastFetchError, experr) {
			t.Errorf("got '%s' expected prefix '%s'", tlf.LastFetchError, experr)
		}
		if tlf.LastSuccessfulFetchAt.Valid {
			t.Errorf("got %t expected false", tlf.LastSuccessfulFetchAt.Valid)
		}
		// Check FeedFetch record
		tlff := dmfr.FeedFetch{}
		testdb.ShouldGet(t, atx, &tlff, `SELECT * FROM feed_fetches WHERE feed_id = ? ORDER BY id DESC LIMIT 1`, feed.ID)
		assert.Equal(t, 404, tlff.ResponseCode.Int, "did not get expected feed_fetch response code")
		assert.Equal(t, false, tlff.Success, "did not get expected feed_fetch success")
		return nil
	})
}
