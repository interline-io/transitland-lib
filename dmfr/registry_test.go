package dmfr

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
)

func TestLoadAndParseRegistry_from_file(t *testing.T) {
	parsedContents, err := LoadAndParseRegistry(testutil.RelPath("test/data/dmfr/example.json"))
	if err != nil {
		log.Fatal(err)
		t.Error(err)
	}
	if len(parsedContents.Feeds) != 2 {
		t.Error("didn't load all 2 feeds")
	}
	if parsedContents.LicenseSpdxIdentifier != "CC0-1.0" {
		t.Error("LicenseSpdxIdentifier is not equal to 'CC0-1.0'")
	}
	if len(parsedContents.Operators) != 1 {
		t.Errorf("got %d operators in feed, expected %d", len(parsedContents.Operators), 1)
	}
}

func TestParseOperators(t *testing.T) {
	parsedContents, err := LoadAndParseRegistry(testutil.RelPath("test/data/dmfr/example.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(parsedContents.Operators) != 1 {
		t.Fatalf("got %d top level operators, expected %d", len(parsedContents.Operators), 1)
	}
	o := parsedContents.Operators[0]
	if o.OnestopID.String != "test" {
		t.Errorf("got '%s' onestop_id, expected '%s'", o.OnestopID.String, "test")
	}
	if len(o.AssociatedFeeds) != 2 {
		t.Fatalf("got %d operator associated feeds, expected %d", len(o.AssociatedFeeds), 2)
	}
	for _, oif := range o.AssociatedFeeds {
		if oif.FeedOnestopID.String == "GT" {
			if oif.GtfsAgencyID.String != "abc" {
				t.Errorf("got '%s' agency_id, expected '%s'", oif.GtfsAgencyID.String, "abc")
			}
		}
	}
}

func TestLoadAndParseRegistry_from_URL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile(testutil.RelPath("test/data/dmfr/example.json"))
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))
	defer ts.Close()
	parsedContents, err := LoadAndParseRegistry(ts.URL)
	if err != nil {
		log.Fatal(err)
		t.Error(err)
	}
	if len(parsedContents.Feeds) != 2 {
		t.Error("didn't load all 2 feeds")
	}
	if parsedContents.LicenseSpdxIdentifier != "CC0-1.0" {
		t.Error("LicenseSpdxIdentifier is not equal to 'CC0-1.0'")
	}
}

func TestParseString(t *testing.T) {
	dmfrString, err := ioutil.ReadFile(testutil.RelPath("test/data/dmfr/example.json"))
	if err != nil {
		t.Error("failed to read sample dmfr")
	}
	feed, _ := ParseString(string(dmfrString))
	if len(feed.Feeds) != 2 {
		t.Error("didn't load all 2 feeds")
	}
	if feed.LicenseSpdxIdentifier != "CC0-1.0" {
		t.Error("LicenseSpdxIdentifier is not equal to 'CC0-1.0'")
	}
}
