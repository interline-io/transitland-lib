package dmfr

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/internal/testdb"
)

func caltrain(atx gtdb.Adapter, url string) int {
	// Create dummy feed
	tlfeed := Feed{}
	tlfeed.FeedID = url
	tlfeed.URLs.StaticCurrent = url
	feedid := testdb.MustInsert(atx, &tlfeed)
	return feedid
}

func TestMainFetchFeed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile("../testdata/example.zip")
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
		expsha := "21e43625117b993c125f4a939973a862e2cbd136"
		url := ts.URL
		feedid := caltrain(atx, ts.URL)
		fv, found, err := MainFetchFeed(atx, feedid, tmpdir)
		if err != nil {
			t.Error(err)
			return nil
		}
		if found {
			t.Errorf("expected new fv")
			return nil
		}
		if fv.SHA1 != expsha {
			t.Errorf("got %s expect %s", fv.SHA1, expsha)
			return nil
		}
		// Check FV
		fv2 := gotransit.FeedVersion{ID: fv.ID}
		testdb.ShouldFind(t, atx, &fv2)
		if fv2.URL != url {
			t.Errorf("got %s expect %s", fv2.URL, url)
		}
		if fv2.FeedID != feedid {
			t.Errorf("got %d expect %d", fv2.FeedID, feedid)
		}
		if fv2.SHA1 != expsha {
			t.Errorf("got %s expect %s", fv2.SHA1, expsha)
		}
		// Check Feed
		tlf := Feed{ID: feedid}
		testdb.ShouldFind(t, atx, &tlf)
		if !tlf.LastSuccessfulFetchAt.Valid {
			t.Errorf("expected non-nil value")
		}
		// Check that we saved the output file
		outfn := filepath.Join(tmpdir, fv.SHA1+".zip")
		info, err := os.Stat(outfn)
		if os.IsNotExist(err) {
			t.Errorf("expected file to exist: %s", outfn)
		}
		expsize := int64(4120)
		if info.Size() != expsize {
			t.Errorf("got %d bytes in file, expected %d", info.Size(), expsize)
		}
		return nil
	})
}

func TestMainFetchFeed_LastFetchError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Status-Code", "404")
		w.Write([]byte("not found"))
	}))
	defer ts.Close()
	testdb.WithAdapterRollback(func(atx gtdb.Adapter) error {
		feedid := caltrain(atx, ts.URL)
		// Fetch
		_, _, err := MainFetchFeed(atx, feedid, "")
		if err != nil {
			t.Error(err)
			return nil
		}
		// Check
		tlf := Feed{}
		tlf.ID = feedid
		testdb.ShouldFind(t, atx, &tlf)
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
		buf, err := ioutil.ReadFile("../testdata/example.zip")
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))
	defer ts.Close()
	testdb.WithAdapterRollback(func(atx gtdb.Adapter) error {
		expsha1 := "21e43625117b993c125f4a939973a862e2cbd136"
		url := ts.URL
		feedid := caltrain(atx, url)
		fv, found, err := FetchAndCreateFeedVersion(atx, feedid, url, time.Now(), "")
		if err != nil {
			t.Error(err)
			return err
		}
		if found {
			t.Error("expected new feed")
			return nil
		}
		if fv.SHA1 != expsha1 {
			t.Errorf("got %s expect %s", fv.SHA1, expsha1)
			return nil
		}
		fv2 := gotransit.FeedVersion{ID: fv.ID}
		testdb.ShouldFind(t, atx, &fv2)
		if fv2.URL != url {
			t.Errorf("got %s expect %s", fv2.URL, url)
		}
		if fv2.FeedID != feedid {
			t.Errorf("got %d expect %d", fv2.FeedID, feedid)
		}
		if fv2.SHA1 != expsha1 {
			t.Errorf("got %s expect %s", fv2.SHA1, expsha1)
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
		fv, _, err := FetchAndCreateFeedVersion(atx, feedid, url, time.Now(), "")
		if err == nil {
			t.Error("expected error")
			return nil
		}
		if fv.ID != 0 {
			t.Errorf("got %d expect %d", fv.ID, 0)
		}
		errmsg := err.Error()
		experr := "file does not exist"
		if !strings.HasPrefix(errmsg, experr) {
			t.Errorf("got '%s' expected prefix '%s'", errmsg, experr)
		}
		return nil
	})
}

func TestFetchAndCreateFeedVersion_Exists(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile("../testdata/example.zip")
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))
	testdb.WithAdapterRollback(func(atx gtdb.Adapter) error {
		expsha1 := "21e43625117b993c125f4a939973a862e2cbd136"
		url := ts.URL
		feedid := caltrain(atx, url)
		fv, found, err := FetchAndCreateFeedVersion(atx, feedid, url, time.Now(), "")
		if err != nil {
			t.Error(err)
		}
		if found {
			t.Error("expected new feed")
		}
		if fv.SHA1 != expsha1 {
			t.Errorf("got %s expect %s", fv.SHA1, expsha1)
		}
		fv2, found2, err2 := FetchAndCreateFeedVersion(atx, feedid, url, time.Now(), "")
		if err2 != nil {
			t.Error(err2)
			return err2
		}
		if !found2 {
			t.Error("expected found feed")
		}
		if fv2.SHA1 != expsha1 {
			t.Errorf("got %s expect %s", fv2.SHA1, expsha1)
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
