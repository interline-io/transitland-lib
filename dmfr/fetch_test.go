package dmfr

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/internal/testdb"
	"github.com/interline-io/gotransit/internal/testutil"
)

var ExampleZip = testutil.ExampleZip

func caltrain(atx gtdb.Adapter, url string) int {
	// Create dummy feed
	tlfeed := Feed{}
	tlfeed.FeedID = url
	tlfeed.URLs.StaticCurrent = url
	feedid := testdb.MustInsert(atx, &tlfeed)
	return feedid
}

func TestDatabaseFetch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile(ExampleZip.URL)
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))
	defer ts.Close()
	testdb.WithAdapterRollback(func(atx gtdb.Adapter) error {
		tmpdir, err := ioutil.TempDir("", "gtfs")
		if err != nil {
			t.Error(err)
			return nil
		}
		defer os.RemoveAll(tmpdir) // clean up
		//
		url := ts.URL
		feedid := caltrain(atx, ts.URL)
		fr, err := DatabaseFetch(atx, FetchOptions{FeedID: feedid, Directory: tmpdir})
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
		fv2 := gotransit.FeedVersion{ID: fr.FeedVersion.ID}
		testdb.ShouldFind(t, atx, &fv2)
		if fv2.URL != url {
			t.Errorf("got %s expect %s", fv2.URL, url)
		}
		if fv2.FeedID != feedid {
			t.Errorf("got %d expect %d", fv2.FeedID, feedid)
		}
		if fv2.SHA1 != ExampleZip.SHA1 {
			t.Errorf("got %s expect %s", fv2.SHA1, ExampleZip.SHA1)
		}
		// Check FeedState
		tlf := FeedState{}
		testdb.ShouldGet(t, atx, &tlf, `SELECT * FROM feed_states WHERE feed_id = ?`, feedid)
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

func TestDatabaseFetch_LastFetchError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Status-Code", "404")
		w.Write([]byte("not found"))
	}))
	defer ts.Close()
	testdb.WithAdapterRollback(func(atx gtdb.Adapter) error {
		tmpdir, err := ioutil.TempDir("", "gtfs")
		if err != nil {
			t.Error(err)
			return nil
		}
		defer os.RemoveAll(tmpdir) // clean up
		feedid := caltrain(atx, ts.URL)
		// Fetch
		_, err = DatabaseFetch(atx, FetchOptions{FeedID: feedid, Directory: tmpdir})
		if err != nil {
			t.Error(err)
			return nil
		}
		// Check FeedState
		tlf := FeedState{}
		testdb.ShouldGet(t, atx, &tlf, `SELECT * FROM feed_states WHERE feed_id = ?`, feedid)
		experr := "file does not exist"
		if tlf.LastFetchError == "" {
			t.Errorf("expected value for LastFetchError")
		}
		if !strings.HasPrefix(tlf.LastFetchError, experr) {
			t.Errorf("got '%s' expected prefix '%s'", tlf.LastFetchError, experr)
		}
		if !tlf.LastSuccessfulFetchAt.Valid {
			t.Errorf("got %t expected false", tlf.LastSuccessfulFetchAt.Valid)
		}
		return nil
	})
}

func TestFetchAndCreateFeedVersion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile(ExampleZip.URL)
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))
	defer ts.Close()
	testdb.WithAdapterRollback(func(atx gtdb.Adapter) error {
		tmpdir, err := ioutil.TempDir("", "gtfs")
		if err != nil {
			t.Error(err)
			return nil
		}
		defer os.RemoveAll(tmpdir) // clean up
		url := ts.URL
		feedid := caltrain(atx, url)
		fr, err := FetchAndCreateFeedVersion(atx, FetchOptions{FeedID: feedid, FeedURL: url, Directory: tmpdir})
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
		fv2 := gotransit.FeedVersion{ID: fr.FeedVersion.ID}
		testdb.ShouldFind(t, atx, &fv2)
		if fv2.URL != url {
			t.Errorf("got %s expect %s", fv2.URL, url)
		}
		if fv2.FeedID != feedid {
			t.Errorf("got %d expect %d", fv2.FeedID, feedid)
		}
		if fv2.SHA1 != ExampleZip.SHA1 {
			t.Errorf("got %s expect %s", fv2.SHA1, ExampleZip.SHA1)
		}
		return nil
	})
}

func TestFetchAndCreateFeedVersion_404(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Status-Code", "404")
		w.Write([]byte("not found"))
	}))
	defer ts.Close()
	testdb.WithAdapterRollback(func(atx gtdb.Adapter) error {
		url := ts.URL
		feedid := caltrain(atx, url)
		fr, err := FetchAndCreateFeedVersion(atx, FetchOptions{FeedID: feedid, FeedURL: url, Directory: ""})
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

func TestFetchAndCreateFeedVersion_Exists(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile(ExampleZip.URL)
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))
	testdb.WithAdapterRollback(func(atx gtdb.Adapter) error {
		url := ts.URL
		feedid := caltrain(atx, url)
		fr, err := FetchAndCreateFeedVersion(atx, FetchOptions{FeedID: feedid, FeedURL: url, Directory: ""})
		if err != nil {
			t.Error(err)
		}
		if fr.FoundSHA1 || fr.FoundDirSHA1 {
			t.Error("expected new feed")
		}
		if fr.FeedVersion.SHA1 != ExampleZip.SHA1 {
			t.Errorf("got %s expect %s", fr.FeedVersion.SHA1, ExampleZip.SHA1)
		}
		fr2, err2 := FetchAndCreateFeedVersion(atx, FetchOptions{FeedID: feedid, FeedURL: url, Directory: ""})
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
