package dmfr

import (
	"testing"
	"time"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/internal/testdb"
	"github.com/interline-io/gotransit/internal/testutil"
)

// Full tests
func TestMainSync(t *testing.T) {
	err := testdb.WithAdapterRollback(func(atx gtdb.Adapter) error {
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
		expect := map[int]bool{}
		for _, i := range found {
			expect[i] = true
		}
		tlfeeds := []Feed{}
		testdb.ShouldSelect(t, atx, &tlfeeds, "SELECT * FROM current_feeds WHERE deleted_at IS NULL")
		if len(tlfeeds) != len(expect) {
			t.Errorf("got %d feeds, expect %d", len(tlfeeds), len(expect))
		}
		for _, tlfeed := range tlfeeds {
			if _, ok := expect[tlfeed.ID]; !ok {
				t.Errorf("did not find feed %s", tlfeed.FeedID)
			}
		}
		hf := Feed{}
		testdb.ShouldGet(t, atx, &hf, "SELECT * FROM current_feeds WHERE onestop_id = ?", "caltrain")
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
	err := testdb.WithAdapterRollback(func(atx gtdb.Adapter) error {
		// Create existing feed
		fetchtime := gotransit.OptionalTime{Time: time.Now().UTC(), Valid: true}
		experr := "checking preserved values"
		exposid := "f-c20-trimet"
		tlfeed := Feed{}
		tlfeed.URLs.StaticCurrent = "http://example.com"
		tlfeed.FeedNamespaceID = "o-example-nsid"
		tlfeed.FeedID = exposid
		tlfeed.LastFetchError = experr
		tlfeed.LastFetchedAt = fetchtime
		tlfeed.LastImportedAt = fetchtime
		tlfeed.LastSuccessfulFetchAt = fetchtime
		var err error
		tlfeed.ID = testdb.ShouldInsert(t, atx, &tlfeed)
		if err != nil {
			t.Error(err)
		}
		// Import
		regs := []string{"../testdata/dmfr/rtfeeds.dmfr.json"}
		if _, err = MainSync(atx, regs); err != nil {
			t.Error(err)
		}
		// Check Updated values
		testdb.ShouldFind(t, atx, &tlfeed)
		expurl := "https://developer.trimet.org/schedule/gtfs.zip"
		if tlfeed.URLs.StaticCurrent != expurl {
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

// Unit tests

func TestImportFeed(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		err := testdb.WithAdapterRollback(func(atx gtdb.Adapter) error {
			rfeed := Feed{}
			rfeed.FeedID = "caltrain"
			rfeed.Spec = "gtfs"
			rfeed.URLs.StaticCurrent = "http://example.com/caltrain.zip"
			rfeed.License.UseWithoutAttribution = "yes"
			rfeed.Authorization.ParamName = "test"
			rfeed.Languages = FeedLanguages{"en": "ok"}
			feedid, found, err := ImportFeed(atx, rfeed)
			if err != nil {
				t.Error(err)
			}
			if found {
				t.Errorf("expected new feed")
			}
			//
			dfeed := Feed{}
			testdb.ShouldGet(t, atx, &dfeed, "SELECT * FROM current_feeds WHERE id = ?", feedid)
			if a, b := dfeed.FeedID, rfeed.FeedID; a != b {
				t.Errorf("got %s expect %s", a, b)
			}
			if a, b := dfeed.Spec, rfeed.Spec; a != b {
				t.Errorf("got %s expect %s", a, b)
			}
			if a, b := dfeed.URLs.StaticCurrent, rfeed.URLs.StaticCurrent; a != b {
				t.Errorf("got %s expect %s", a, b)
			}
			if a, b := dfeed.License.UseWithoutAttribution, rfeed.License.UseWithoutAttribution; a != b {
				t.Errorf("got %s expect %s", a, b)
			}
			if a, b := dfeed.Authorization.ParamName, rfeed.Authorization.ParamName; a != b {
				t.Errorf("got %s expect %s", a, b)
			}
			if a, b := dfeed.Languages["en"], rfeed.Languages["en"]; a != b || a != "ok" {
				t.Errorf("got %s expect %s", a, b)
			}
			return nil
		})
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("Update", func(t *testing.T) {
		err := testdb.WithAdapterRollback(func(atx gtdb.Adapter) error {
			rfeed := Feed{}
			rfeed.FeedID = "caltrain"
			feedid, found, err := ImportFeed(atx, rfeed)
			if err != nil {
				t.Error(err)
			}
			if found == true {
				t.Errorf("expected new feed")
			}
			// Reload
			testdb.ShouldGet(t, atx, &rfeed, "SELECT * FROM current_feeds WHERE id = ?", feedid)
			//
			dfeed := Feed{}
			dfeed.FeedID = "caltrain"
			feedid2, found2, err2 := ImportFeed(atx, dfeed)
			if err2 != nil {
				t.Error(err)
			}
			if feedid2 != feedid {
				t.Errorf("got %d expect %d", feedid2, feedid)
			}
			if found2 == false {
				t.Errorf("expected updated feed")
			}
			// Reload
			testdb.ShouldGet(t, atx, &dfeed, "SELECT * FROM current_feeds WHERE id = ?", feedid2)
			// Test
			if a, b := dfeed.FeedID, rfeed.FeedID; a != b {
				t.Errorf("got %s expect %s", a, b)
			}
			if a, b := dfeed.CreatedAt, rfeed.CreatedAt; !a.Equal(b) {
				t.Errorf("expected %s got %s", a, b)
			}
			if a, b := dfeed.UpdatedAt, rfeed.UpdatedAt; !a.After(b) {
				t.Errorf("expected updated_at %s to be greater than %s", a, b)
			}
			if !(dfeed.DeletedAt.IsZero() || dfeed.DeletedAt.Valid) {
				t.Errorf("expected deleted_at to be null, got %s %t", dfeed.DeletedAt.Time, dfeed.DeletedAt.Valid)
			}
			return nil
		})
		if err != nil {
			t.Error(err)
		}
	})

}

func TestHideUnseedFeeds(t *testing.T) {
	err := testdb.WithAdapterRollback(func(atx gtdb.Adapter) error {
		feedids := []string{"caltrain", "seen"}
		fids := []int{}
		for _, feedid := range feedids {
			f := caltrain(atx, feedid)
			fids = append(fids, f)
		}
		expseen := fids[0:1]
		expunseen := fids[1:]
		count, err := HideUnseedFeeds(atx, expseen)
		if err != nil {
			t.Error(err)
		}
		if count != len(expunseen) {
			t.Errorf("got %d expect %d", count, len(expunseen))
		}
		// check soft deleted
		seen := []int{}
		testdb.ShouldSelect(t, atx, &seen, "SELECT id FROM current_feeds WHERE deleted_at IS NULL")
		if !testutil.CompareSliceInt(seen, expseen) {
			t.Errorf("%v != %v", seen, expseen)
		}
		unseen := []int{}
		testdb.ShouldSelect(t, atx, &unseen, "SELECT id FROM current_feeds WHERE deleted_at IS NOT NULL")
		if !testutil.CompareSliceInt(unseen, expunseen) {
			t.Errorf("%v != %v", unseen, expunseen)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}
