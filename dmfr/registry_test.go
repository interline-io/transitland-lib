package dmfr

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestLoadAndParseRegistry_from_file(t *testing.T) {
	parsedContents, err := LoadAndParseRegistry(testutil.RelPath("test/data/dmfr/example.json"))
	if err != nil {
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
		t.Error(err)
	}
	if len(parsedContents.Feeds) != 2 {
		t.Error("didn't load all 2 feeds")
	}
	if parsedContents.LicenseSpdxIdentifier != "CC0-1.0" {
		t.Error("LicenseSpdxIdentifier is not equal to 'CC0-1.0'")
	}
}

func TestLoadAndParseRegistry_Secrets(t *testing.T) {
	parsedContents, err := LoadAndParseRegistry(testutil.RelPath("test/data/dmfr/secrets.json"))
	if err != nil {
		t.Error(err)
	}
	if len(parsedContents.Secrets) != 4 {
		t.Errorf("got %d secrets, expected %d", len(parsedContents.Secrets), 4)
	}
}

func TestImplicitOperatorInFeed(t *testing.T) {
	reg, err := LoadAndParseRegistry(testutil.RelPath("test/data/dmfr/embedded.json"))
	if err != nil {
		t.Fatal(err)
	}
	tcs := []struct {
		name     string
		feedname string
		feedOps  []string
		opname   string
		opFeeds  []string
	}{
		{"no operators", "f-other~feed", []string{}, "", []string{}}, // no operators??
		{"with implicit", "f-with~implicit", []string{"o-with~implicit"}, "o-with~implicit", []string{"f-with~implicit"}},
		{"with explicit", "f-with~explicit", []string{"o-with~explicit"}, "o-with~explicit", []string{"f-with~explicit"}},
		{"with explicit mixed", "f-with~explicit~mixed", []string{"o-test"}, "o-test", []string{"f-other~feed", "f-with~explicit~mixed"}},

		{"toplevel no feed", "", []string{}, "o-toplevel~nofeed", []string{}},
		{"toplevel onefeed", "f-test2", []string{"o-toplevel~onefeed"}, "o-toplevel~onefeed", []string{"f-test2"}},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			testMatch := false
			matchFeeds := map[string][]string{}
			for _, op := range reg.Operators {
				if op.OnestopID.String != tc.opname {
					continue
				}
				testMatch = true
				foundFeeds := []string{}
				for _, feed := range op.AssociatedFeeds {
					foundFeeds = append(foundFeeds, feed.FeedOnestopID.String)
					matchFeeds[feed.FeedOnestopID.String] = append(matchFeeds[feed.FeedOnestopID.String], op.OnestopID.String)
				}
				assert.ElementsMatchf(t, tc.opFeeds, foundFeeds, "operator %s did not match expected feeds", tc.opname)
			}
			if tc.feedname != "" {
				testMatch = true
				assert.ElementsMatchf(t, tc.feedOps, matchFeeds[tc.feedname], "feed %s did not match expected operators", tc.feedname)
			}
			if !testMatch {
				t.Errorf("no matching tests")
			}
		})
	}
}
