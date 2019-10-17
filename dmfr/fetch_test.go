package dmfr

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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

func caltrain(atx gtdb.Adapter, url string) int {
	// Create dummy feed
	tlfeed := Feed{}
	tlfeed.FeedID = url
	tlfeed.URL = url
	var err error
	// err := atx.Get(&tlfeed, "SELECT * FROM current_feeds WHERE onestop_id = ?", "caltrain")
	// if err == sql.ErrNoRows {
	tlfeed.ID, err = atx.Insert(&tlfeed)
	// }
	if err != nil {
		panic(err)
	}
	return tlfeed.ID
}

// func TestMainFetchFeed(t *testing.T) {
// 	withDB(func(tx *sqlx.DB) {
// 		feedid, feedurl := caltrain(tx)
// 		fvid, err := MainFetchFeed(tx, feedid)
// 		if err != nil {
// 			t.Error(err)
// 			return
// 		}
// 		// Check FV
// 		fv := gotransit.FeedVersion{}
// 		if err := tx.Find(&fv, fvid).Error; err != nil {
// 			t.Error(err)
// 		}
// 		if fv.URL != feedurl {
// 			t.Errorf("got %s expect %s", fv.URL, feedurl)
// 		}
// 		if fv.FeedID != feedid {
// 			t.Errorf("got %d expect %d", fv.FeedID, feedid)
// 		}
// 		expsha := "2e2142145d772ecbb523b2c9978641ccc4a59ea6"
// 		if fv.SHA1 != expsha {
// 			t.Errorf("got %s expect %s", fv.SHA1, expsha)
// 		}
// 		// Check Feed
// 		tlf := dbFeed{}
// 		tlf.ID = feedid
// 		tx.Find(&tlf)
// 		if tlf.LastSuccessfulFetchAt == nil {
// 			t.Errorf("expected non-nil value")
// 		}
// 	})
// }

// func TestMainFetchFeed_LastFetchError(t *testing.T) {
// 	withDB(func(tx *sqlx.DB) {
// 		feedid, _ := caltrain(tx)
// 		tlf := dbFeed{}
// 		tlf.ID = feedid
// 		tx.Model(&tlf).Update("url", "http://localhost:8000/invalid.zip")
// 		if _, err := MainFetchFeed(tx, feedid); err != nil {
// 			t.Error(err)
// 			return
// 		}
// 		tx.Find(&tlf)
// 		experr := "required file not present"
// 		if !strings.HasPrefix(tlf.LastFetchError, experr) {
// 			t.Errorf("got '%s' expected prefix '%s'", tlf.LastFetchError, experr)
// 		}
// 		if tlf.LastSuccessfulFetchAt != nil {
// 			t.Errorf("got %s expected nil", tlf.LastSuccessfulFetchAt)
// 		}
// 	})
// }

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
		url := ts.URL
		feedid := caltrain(atx, url)
		fvid, err := FetchAndCreateFeedVersion(atx, feedid, url, time.Now())
		if err != nil {
			t.Error(err)
			return err
		}
		fv := gotransit.FeedVersion{}
		fv.ID = fvid
		if err := atx.Find(&fv); err != nil {
			panic(err)
		}
		if fv.URL != url {
			t.Errorf("got %s expect %s", fv.URL, url)
		}
		if fv.FeedID != feedid {
			t.Errorf("got %d expect %d", fv.FeedID, feedid)
		}
		return nil
	})
}

// func TestFetchAndCreateFeedVersion_404(t *testing.T) {
// 	withDB(func(tx *sqlx.DB) {
// 		feedid, _ := caltrain(tx)
// 		url := "http://localhost:8000/notfound.zip"
// 		fvid, err := FetchAndCreateFeedVersion(tx, feedid, url, time.Now())
// 		if err == nil {
// 			t.Error("expected error")
// 			return
// 		}
// 		if fvid != 0 {
// 			t.Errorf("got %d expect %d", fvid, 0)
// 		}
// 		errmsg := err.Error()
// 		experr := "file does not exist"
// 		if !strings.HasPrefix(errmsg, experr) {
// 			t.Errorf("got '%s' expected prefix '%s'", errmsg, experr)
// 		}
// 	})
// }

// func TestFetchAndCreateFeedVersion_Exists(t *testing.T) {
// 	withDB(func(tx *sqlx.DB) {
// 		feedid, url := caltrain(tx)
// 		fvid, err := FetchAndCreateFeedVersion(tx, feedid, url, time.Now())
// 		if err != nil {
// 			t.Error(err)
// 			return
// 		}
// 		fvid2, err2 := FetchAndCreateFeedVersion(tx, feedid, url, time.Now())
// 		if err2 != nil {
// 			t.Error(err)
// 			return
// 		}
// 		if fvid == 0 {
// 			t.Error("expected non-zero value")
// 		}
// 		if fvid != fvid2 {
// 			t.Errorf("got %d expected %d", fvid, fvid2)
// 		}
// 		fv := gotransit.FeedVersion{}
// 		tx.Find(&fv, fvid)
// 		if fv.FeedID != feedid {
// 			t.Errorf("got %d expected %d", fv.FeedID, feedid)
// 		}
// 	})
// }
