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
)

var ExampleZip = testutil.ExampleZip

func TestDatabaseFetch(t *testing.T) {
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
		fr, err := DatabaseFetch(atx, Options{FeedID: feed.FeedID, Directory: tmpdir})
		if err != nil {
			t.Error(err)
			return nil
		}
		if fr.FoundSHA1 || fr.FoundDirSHA1 {
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
		// Check that we saved the output file
		outfn := filepath.Join(tmpdir, fr.FeedVersion.SHA1+".zip")
		info, err := os.Stat(outfn)
		if os.IsNotExist(err) {
			t.Errorf("expected file to exist: %s", outfn)
		}
		expsize := int64(ExampleZip.Size)
		if info.Size() != expsize {
			t.Errorf("got %d bytes in file, expected %d", info.Size(), expsize)
		}
		return nil
	})
}

func TestDatabaseFetchCreateFeed(t *testing.T) {
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
		fr, err := DatabaseFetch(atx, Options{FeedID: "caltrain", FeedURL: ts.URL, FeedCreate: true, Directory: tmpdir})
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

func TestDatabaseFetch_LastFetchError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Status-Code", "404")
		w.Write([]byte("not found"))
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
		_, err = DatabaseFetch(atx, Options{FeedID: feed.FeedID, Directory: tmpdir})
		if err != nil {
			t.Error(err)
			return nil
		}
		// Check FeedState
		tlf := dmfr.FeedState{}
		testdb.ShouldGet(t, atx, &tlf, `SELECT * FROM feed_states WHERE feed_id = ?`, feed.ID)
		experr := "file does not exist"
		if tlf.LastFetchError == "" {
			t.Errorf("expected value for LastFetchError")
		}
		if !strings.HasPrefix(tlf.LastFetchError, experr) {
			t.Errorf("got '%s' expected prefix '%s'", tlf.LastFetchError, experr)
		}
		if tlf.LastSuccessfulFetchAt.Valid {
			t.Errorf("got %t expected false", tlf.LastSuccessfulFetchAt.Valid)
		}
		return nil
	})
}

func Test_fetchAndCreateFeedVersion(t *testing.T) {
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
		url := ts.URL
		feed := testdb.CreateTestFeed(atx, url)
		fr, err := fetchAndCreateFeedVersion(atx, feed, Options{FeedURL: url, Directory: tmpdir})
		if err != nil {
			t.Error(err)
			return err
		}
		if fr.FoundSHA1 || fr.FoundDirSHA1 {
			t.Error("expected new feed")
			return nil
		}
		if fr.FeedVersion.SHA1 != ExampleZip.SHA1 {
			t.Errorf("got %s expect %s", fr.FeedVersion.SHA1, ExampleZip.SHA1)
			return nil
		}
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
		return nil
	})
}

func Test_fetchAndCreateFeedVersion_404(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Status-Code", "404")
		w.Write([]byte("not found"))
	}))
	defer ts.Close()
	testdb.WithAdapterRollback(func(atx tldb.Adapter) error {
		url := ts.URL
		feed := testdb.CreateTestFeed(atx, url)
		fr, err := fetchAndCreateFeedVersion(atx, feed, Options{FeedURL: url, Directory: ""})
		if err != nil {
			t.Error(err)
			return err
		}
		if fr.FetchError == nil {
			t.Error("expected error")
			return nil
		}
		if fr.FeedVersion.ID != 0 {
			t.Errorf("got %d expect %d", fr.FeedVersion.ID, 0)
		}
		errmsg := fr.FetchError.Error()
		experr := "file does not exist"
		if !strings.HasPrefix(errmsg, experr) {
			t.Errorf("got '%s' expected prefix '%s'", errmsg, experr)
		}
		return nil
	})
}

func Test_fetchAndCreateFeedVersion_Exists(t *testing.T) {
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
		fr, err := fetchAndCreateFeedVersion(atx, feed, Options{FeedURL: url, Directory: ""})
		if err != nil {
			t.Error(err)
		}
		if fr.FoundSHA1 || fr.FoundDirSHA1 {
			t.Error("expected new feed")
		}
		if fr.FeedVersion.SHA1 != ExampleZip.SHA1 {
			t.Errorf("got %s expect %s", fr.FeedVersion.SHA1, ExampleZip.SHA1)
		}
		fr2, err2 := fetchAndCreateFeedVersion(atx, feed, Options{FeedURL: url, Directory: ""})
		if err2 != nil {
			t.Error(err2)
			return err2
		}
		if !(fr2.FoundSHA1 || fr.FoundDirSHA1) {
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
