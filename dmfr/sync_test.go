package dmfr

import (
	"testing"

	"github.com/interline-io/gotransit/gtdb"
)

func fail(t *testing.T, err error) {
	if err != nil {
		panic(err)
		t.Error(err)
		t.FailNow()
	}
}

func TestMainSync(t *testing.T) {
	WithAdapterTx(func(atx gtdb.Adapter) error {
		// Create a feed we will check is soft-deleted
		caltrain(atx, "caltrain")
		// Import
		regs := []string{
			"../testdata/dmfr/rtfeeds.dmfr.json",
			"../testdata/dmfr/bayarea.dmfr.json",
		}
		found, err := MainSync(atx, regs)
		fail(t, err)
		// Check results
		expect := map[string]bool{}
		for _, i := range found {
			expect[i] = true
		}
		tlfeeds := []Feed{}
		fail(t, atx.Select(&tlfeeds, "SELECT * FROM current_feeds WHERE deleted_at IS NULL"))
		if len(tlfeeds) != len(expect) {
			t.Errorf("got %d feeds, expect %d", len(tlfeeds), len(expect))
		}
		for _, tlfeed := range tlfeeds {
			if _, ok := expect[tlfeed.FeedID]; !ok {
				t.Errorf("did not find feed %s", tlfeed.FeedID)
			}
		}
		hf := Feed{}
		fail(t, atx.Get(&hf, "SELECT * FROM current_feeds WHERE onestop_id = ?", "caltrain"))
		if !hf.DeletedAt.Valid {
			t.Errorf("expected DeletedAt to be non-nil")
		}
		return nil
	})
}

// func TestMainSync_Update(t *testing.T) {
// 	withDB(func(tx *gorm.DB) {
// 		// Create existing feed
// 		fetchtime := time.Now().UTC()
// 		experr := "checking preserved values"
// 		exposid := "f-c20-trimet"
// 		tlfeed := &dbFeed{}
// 		tlfeed.URL = "http://example.com"
// 		tlfeed.FeedNamespaceID = "o-example-nsid"
// 		tlfeed.OnestopID = exposid
// 		tlfeed.LastFetchError = experr
// 		tlfeed.LastFetchedAt = &fetchtime
// 		tlfeed.LastImportedAt = &fetchtime
// 		tlfeed.LastSuccessfulFetchAt = &fetchtime
// 		// expactive := 123
// 		// tlfeed.ActiveFeedVersionID = expactive
// 		fail(t, tx.Create(&tlfeed).Error)
// 		// Import
// 		regs := []string{"testdata/rtfeeds.dmfr.json"}
// 		_, err := MainSync(tx, regs)
// 		fail(t, err)
// 		// Check
// 		if err := tx.Find(&tlfeed).Error; err != nil {
// 			t.Error(err)
// 		}
// 		// Check Updated values
// 		expurl := "https://developer.trimet.org/schedule/gtfs.zip"
// 		if tlfeed.URL != expurl {
// 			t.Errorf("got '%s' expected '%s'", tlfeed.URL, expurl)
// 		}
// 		// TODO: not sure why failing
// 		// expnsid := "o-c20-trimet"
// 		// if tlfeed.FeedNamespaceID != expnsid {
// 		// 	t.Errorf("got '%s' expected '%s'", tlfeed.FeedNamespaceID, expnsid)
// 		// }
// 		// Check Preserved values
// 		if tlfeed.OnestopID != exposid {
// 			t.Errorf("got %s expected %s", tlfeed.OnestopID, exposid)
// 		}
// 		if tlfeed.LastFetchError != experr {
// 			t.Errorf("got %s expected %s", tlfeed.LastFetchError, experr)
// 		}
// 		if !tlfeed.LastFetchedAt.Equal(fetchtime) {
// 			t.Errorf("got %s expected %s", tlfeed.LastFetchedAt, fetchtime)
// 		}
// 		if !tlfeed.LastImportedAt.Equal(fetchtime) {
// 			t.Errorf("got %s expected %s", tlfeed.LastImportedAt, fetchtime)
// 		}
// 		if !tlfeed.LastSuccessfulFetchAt.Equal(fetchtime) {
// 			t.Errorf("got %s expected %s", tlfeed.LastSuccessfulFetchAt, fetchtime)
// 		}
// 		// if tlfeed.ActiveFeedVersionID != expactive {
// 		// 	t.Errorf("got %d expected %d", tlfeed.ActiveFeedVersionID, expactive)
// 		// }
// 	})
// }
