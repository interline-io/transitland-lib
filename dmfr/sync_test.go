package dmfr

import (
	"testing"
	"time"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/gtdb"
)

func TestMainSync(t *testing.T) {
	err := WithAdapterRollback(func(atx gtdb.Adapter) error {
		// Create a feed we will check is soft-deleted
		caltrain(atx, "caltrain")
		// Import
		regs := []string{
			"../testdata/dmfr/rtfeeds.dmfr.json",
			"../testdata/dmfr/bayarea.dmfr.json",
		}
		found, err := MainSync(atx, regs)
		if err != nil {
			t.Error(err)
		}
		// Check results
		expect := map[string]bool{}
		for _, i := range found {
			expect[i] = true
		}
		tlfeeds := []Feed{}
		if err := atx.Select(&tlfeeds, "SELECT * FROM current_feeds WHERE deleted_at IS NULL"); err != nil {
			t.Error(err)
		}
		if len(tlfeeds) != len(expect) {
			t.Errorf("got %d feeds, expect %d", len(tlfeeds), len(expect))
		}
		for _, tlfeed := range tlfeeds {
			if _, ok := expect[tlfeed.FeedID]; !ok {
				t.Errorf("did not find feed %s", tlfeed.FeedID)
			}
		}
		hf := Feed{}
		if err := atx.Get(&hf, "SELECT * FROM current_feeds WHERE onestop_id = ?", "caltrain"); err != nil {
			t.Error(err)
		}
		if !hf.DeletedAt.Valid {
			t.Errorf("expected DeletedAt to be non-nil")
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}

func TestMainSync_Update(t *testing.T) {
	err := WithAdapterRollback(func(atx gtdb.Adapter) error {
		// Create existing feed
		fetchtime := gotransit.OptionalTime{Time: time.Now().UTC(), Valid: true}
		experr := "checking preserved values"
		exposid := "f-c20-trimet"
		tlfeed := Feed{}
		tlfeed.URL = "http://example.com"
		tlfeed.FeedNamespaceID = "o-example-nsid"
		tlfeed.FeedID = exposid
		tlfeed.LastFetchError = experr
		tlfeed.LastFetchedAt = fetchtime
		tlfeed.LastImportedAt = fetchtime
		tlfeed.LastSuccessfulFetchAt = fetchtime
		var err error
		tlfeed.ID, err = atx.Insert(&tlfeed)
		if err != nil {
			t.Error(err)
		}
		// Import
		regs := []string{"../testdata/dmfr/rtfeeds.dmfr.json"}
		if _, err = MainSync(atx, regs); err != nil {
			t.Error(err)
		}
		// Check
		if err := atx.Find(&tlfeed); err != nil {
			t.Error(err)
		}
		// Check Updated values
		expurl := "https://developer.trimet.org/schedule/gtfs.zip"
		if tlfeed.URL != expurl {
			t.Errorf("got '%s' expected '%s'", tlfeed.URL, expurl)
		}
		expnsid := "o-c20-trimet"
		if tlfeed.FeedNamespaceID != expnsid {
			t.Errorf("got '%s' expected '%s'", tlfeed.FeedNamespaceID, expnsid)
		}
		// Check Preserved values
		if tlfeed.FeedID != exposid {
			t.Errorf("got %s expected %s", tlfeed.FeedID, exposid)
		}
		if tlfeed.LastFetchError != experr {
			t.Errorf("got %s expected %s", tlfeed.LastFetchError, experr)
		}
		if !tlfeed.LastFetchedAt.Time.Equal(fetchtime.Time) {
			t.Errorf("got %s expected %s", tlfeed.LastFetchedAt.Time, fetchtime.Time)
		}
		if !tlfeed.LastImportedAt.Time.Equal(fetchtime.Time) {
			t.Errorf("got %s expected %s", tlfeed.LastImportedAt.Time, fetchtime.Time)
		}
		if !tlfeed.LastSuccessfulFetchAt.Time.Equal(fetchtime.Time) {
			t.Errorf("got %s expected %s", tlfeed.LastSuccessfulFetchAt.Time, fetchtime.Time)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}
