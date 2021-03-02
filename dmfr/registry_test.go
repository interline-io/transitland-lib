package dmfr

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
)

func Test_LoadAndParseRegistry_from_file(t *testing.T) {
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
}

func Test_LoadAndParseRegistry_from_URL(t *testing.T) {
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

func Test_ParseString(t *testing.T) {
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
