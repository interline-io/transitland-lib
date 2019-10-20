package dmfr

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/gtdb"
)

// WithAdapterRollback runs a callback inside a Tx and then aborts, returns any error from original callback.
func WithAdapterRollback(cb func(gtdb.Adapter) error) error {
	var err error
	cb2 := func(atx gtdb.Adapter) error {
		err = cb(atx)
		return errors.New("rollback")
	}
	WithAdapterTx(cb2)
	return err
}

// WithAdapterTx runs a callback inside a Tx, commits if callback returns nil.
func WithAdapterTx(cb func(gtdb.Adapter) error) error {
	writer, err := gtdb.NewWriter("postgres://localhost/tl?sslmode=disable")
	if err != nil {
		panic(err)
	}
	if err := writer.Open(); err != nil {
		panic(err)
	}
	defer writer.Close()
	return writer.Adapter.Tx(cb)
}

type AdapterIgnoreTx struct {
	gtdb.Adapter
}

// Tx runs in same tx if tx already open, otherwise runs without tx
func (atx *AdapterIgnoreTx) Tx(cb func(gtdb.Adapter) error) error {
	return cb(atx)
}

func caltrain(atx gtdb.Adapter, url string) int {
	// Create dummy feed
	tlfeed := Feed{}
	tlfeed.FeedID = url
	tlfeed.URL = url
	feedid, err := atx.Insert(&tlfeed)
	if err != nil {
		panic(err)
	}
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
	WithAdapterRollback(func(atx gtdb.Adapter) error {
		url := ts.URL
		feedid := caltrain(atx, ts.URL)
		fvid, _, _, err := MainFetchFeed(atx, feedid)
		if err != nil {
			t.Error(err)
			return nil
		}
		// Check FV
		fv := gotransit.FeedVersion{}
		fv.ID = fvid
		if err := atx.Find(&fv); err != nil {
			t.Error(err)
		}
		if fv.URL != url {
			t.Errorf("got %s expect %s", fv.URL, url)
		}
		if fv.FeedID != feedid {
			t.Errorf("got %d expect %d", fv.FeedID, feedid)
		}
		expsha := "21e43625117b993c125f4a939973a862e2cbd136"
		if fv.SHA1 != expsha {
			t.Errorf("got %s expect %s", fv.SHA1, expsha)
		}
		// Check Feed
		tlf := Feed{}
		tlf.ID = feedid
		atx.Find(&tlf)
		if !tlf.LastSuccessfulFetchAt.Valid {
			t.Errorf("expected non-nil value")
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
	WithAdapterRollback(func(atx gtdb.Adapter) error {
		feedid := caltrain(atx, ts.URL)
		// Fetch
		if _, _, _, err := MainFetchFeed(atx, feedid); err != nil {
			t.Error(err)
			return nil
		}
		// Check
		tlf := Feed{}
		tlf.ID = feedid
		atx.Find(&tlf)
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
	WithAdapterRollback(func(atx gtdb.Adapter) error {
		expsha1 := "21e43625117b993c125f4a939973a862e2cbd136"
		url := ts.URL
		feedid := caltrain(atx, url)
		fvid, found, sha1, err := FetchAndCreateFeedVersion(atx, feedid, url, time.Now())
		if err != nil {
			t.Error(err)
			return err
		}
		if found {
			t.Error("expected new feed version")
		}
		if sha1 != expsha1 {
			t.Errorf("got %s expect %s", sha1, expsha1)
		}
		fv := gotransit.FeedVersion{}
		fv.ID = fvid
		if err := atx.Find(&fv); err != nil {
			t.Error(err)
		}
		if fv.URL != url {
			t.Errorf("got %s expect %s", fv.URL, url)
		}
		if fv.FeedID != feedid {
			t.Errorf("got %d expect %d", fv.FeedID, feedid)
		}
		if fv.SHA1 != expsha1 {
			t.Errorf("got %s expect %s", fv.SHA1, expsha1)
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
	WithAdapterRollback(func(atx gtdb.Adapter) error {
		url := ts.URL
		feedid := caltrain(atx, url)
		fvid, found, _, err := FetchAndCreateFeedVersion(atx, feedid, url, time.Now())
		if err == nil {
			t.Error("expected error")
			return nil
		}
		if found {
			t.Error("expected not found")
		}
		if fvid != 0 {
			t.Errorf("got %d expect %d", fvid, 0)
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
	WithAdapterRollback(func(atx gtdb.Adapter) error {
		expsha1 := "21e43625117b993c125f4a939973a862e2cbd136"
		url := ts.URL
		feedid := caltrain(atx, url)
		fvid, found, sha1, err := FetchAndCreateFeedVersion(atx, feedid, url, time.Now())
		if err != nil {
			t.Error(err)
			return nil
		}
		if found {
			t.Error("expected not found")
		}
		if sha1 != expsha1 {
			t.Errorf("got %s expect %s", sha1, expsha1)
		}
		fvid2, found2, sha2, err2 := FetchAndCreateFeedVersion(atx, feedid, url, time.Now())
		if err2 != nil {
			t.Error(err2)
			return err2
		}
		if !found2 {
			t.Error("expected to find existing feed version")
		}
		if sha2 != expsha1 {
			t.Errorf("got %s expect %s", sha2, expsha1)
		}
		if fvid == 0 {
			t.Error("expected non-zero value")
		}
		if fvid != fvid2 {
			t.Errorf("got %d expected %d", fvid, fvid2)
		}
		fv := gotransit.FeedVersion{}
		fv.ID = fvid
		atx.Find(&fv)
		if fv.FeedID != feedid {
			t.Errorf("got %d expected %d", fv.FeedID, feedid)
		}
		if fv.SHA1 != expsha1 {
			t.Errorf("got %s expect %s", fv.SHA1, expsha1)
		}
		return nil
	})
}
