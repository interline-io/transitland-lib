package dmfr

import (
	"testing"
	"time"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/gtdb"
	"github.com/jmoiron/sqlx"
)

func withDB(cb func(db *sqlx.Tx)) {
	writer, err := gtdb.NewWriter("postgres://localhost/tl?sslmode=disable")
	if err != nil {
		panic(err)
	}
	if err := writer.Open(); err != nil {
		panic(err)
	}
	defer writer.Close()
	tx, err := writer.Adapter.DBX().Beginx()
	if err != nil {
		panic(err)
	}
	cb(tx)
	tx.Rollback()
}

func caltrain(tx *sqlx.DB) (int, string) {
	// Create dummy feed
	tlfeed := &Feed{}
	tlfeed.OnestopID = "caltrain"
	tlfeed.Spec = "gtfs"
	tlfeed.URL = "http://localhost:8000/CT-GTFS.zip"
	if err := tx.Where("onestop_id = ?", "caltrain").FirstOrCreate(&tlfeed).Error; err != nil {
		panic(err)
	}
	return tlfeed.ID, tlfeed.URL
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
	withDB(func(tx *sqlx.DB) {
		feedid, url := caltrain(tx)
		fvid, err := FetchAndCreateFeedVersion(tx, feedid, url, time.Now())
		if err != nil {
			t.Error(err)
			return
		}
		fv := gotransit.FeedVersion{}
		tx.Find(&fv, fvid)
		if fv.URL != url {
			t.Errorf("got %s expect %s", fv.URL, url)
		}
		if fv.FeedID != feedid {
			t.Errorf("got %d expect %d", fv.FeedID, feedid)
		}
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
