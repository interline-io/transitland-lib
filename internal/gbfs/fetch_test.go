package gbfs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/interline-io/transitland-lib/testdata"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

func TestSystemFileUnmarshalJSON(t *testing.T) {
	t.Run("gbfs 2.x language-keyed data", func(t *testing.T) {
		body := []byte(`{"data":{"en":{"feeds":[{"name":"system_information","url":"http://example.com/system_information.json"}]}}}`)
		var sf SystemFile
		if err := json.Unmarshal(body, &sf); err != nil {
			t.Fatal(err)
		}
		if assert.Contains(t, sf.Data, "en") && assert.NotNil(t, sf.Data["en"]) {
			assert.Len(t, sf.Data["en"].Feeds, 1)
			assert.Equal(t, "system_information", sf.Data["en"].Feeds[0].Name.Val)
		}
	})
	t.Run("gbfs 3.x flat data", func(t *testing.T) {
		body := []byte(`{"data":{"feeds":[{"name":"system_information","url":"http://example.com/system_information.json"}]}}`)
		var sf SystemFile
		if err := json.Unmarshal(body, &sf); err != nil {
			t.Fatal(err)
		}
		if assert.Contains(t, sf.Data, "") && assert.NotNil(t, sf.Data[""]) {
			assert.Len(t, sf.Data[""].Feeds, 1)
			assert.Equal(t, "system_information", sf.Data[""].Feeds[0].Name.Val)
		}
	})
	t.Run("empty data object", func(t *testing.T) {
		var sf SystemFile
		if err := json.Unmarshal([]byte(`{"data":{}}`), &sf); err != nil {
			t.Fatal(err)
		}
		assert.Empty(t, sf.Data)
	})
	t.Run("missing data key", func(t *testing.T) {
		var sf SystemFile
		if err := json.Unmarshal([]byte(`{}`), &sf); err != nil {
			t.Fatal(err)
		}
		assert.Empty(t, sf.Data)
	})
	t.Run("malformed 2.x language value is not silently treated as 3.x", func(t *testing.T) {
		var sf SystemFile
		err := json.Unmarshal([]byte(`{"data":{"en":"bad"}}`), &sf)
		assert.Error(t, err)
	})
}

func TestGbfsFetch(t *testing.T) {
	ts := httptest.NewServer(NewTestGbfsServer("en", testdata.Path("server/gbfs")))
	defer ts.Close()
	opts := Options{}
	opts.FeedURL = fmt.Sprintf("%s/%s", ts.URL, "gbfs.json")
	opts.AllowHTTPFetchUnfiltered = true
	feed, _, err := Fetch(context.Background(), nil, opts)
	if err != nil {
		t.Fatal(err)
	}
	if assert.NotNil(t, feed) {
		assert.Equal(t, "Bay Wheels", feed.SystemInformation.Name.Val)
	}
}

// A feed publishes the same system once per language, and one is fetched: the
// first in order whose files load, whatever order discovery's map comes back in.
func TestFetch_FirstLanguage(t *testing.T) {
	files := httptest.NewServer(NewTestGbfsServer("en", testdata.Path("server/gbfs")))
	defer files.Close()
	resp, err := http.Get(files.URL + "/gbfs.json")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var listed SystemFile
	if err := json.NewDecoder(resp.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}

	// Every language lists the test system's files, with a system_information
	// of its own naming the language. "ca" sorts first but has none, so it is
	// passed over.
	mux := http.NewServeMux()
	discovery := httptest.NewServer(mux)
	defer discovery.Close()
	sf := SystemFile{Data: map[string]*SystemFeeds{}}
	for _, lang := range []string{"nl", "de", "fr", "it", "es", "pt", "ja", "zh", "ko", "sv", "da", "ca"} {
		feeds := &SystemFeeds{}
		for _, f := range listed.Data["en"].Feeds {
			url := f.URL
			if f.Name.Val == "system_information" {
				url = tt.NewString(discovery.URL + "/" + lang + "/system_information.json")
			}
			feeds.Feeds = append(feeds.Feeds, &SystemFeed{Name: f.Name, URL: url})
		}
		sf.Data[lang] = feeds
		if lang == "ca" {
			continue
		}
		mux.HandleFunc("/"+lang+"/system_information.json", func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(SystemInformationFile{Data: &SystemInformation{Language: tt.NewString(lang)}})
		})
	}
	mux.HandleFunc("/gbfs.json", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(sf)
	})

	opts := Options{}
	opts.FeedURL = discovery.URL + "/gbfs.json"
	opts.AllowHTTPFetchUnfiltered = true
	// Map order varies between fetches, so a choice that follows it shows up.
	for range 3 {
		feed, _, err := Fetch(context.Background(), nil, opts)
		if err != nil {
			t.Fatal(err)
		}
		if assert.NotNil(t, feed) {
			assert.Equal(t, "da", feed.SystemInformation.Language.Val)
		}
	}
}
