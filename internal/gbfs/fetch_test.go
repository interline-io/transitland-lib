package gbfs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/interline-io/transitland-lib/request"
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
		assert.Equal(t, "Bay Wheels", feed.SystemInformation.Name.Default())
	}
}

// A 1.x/2.x feed's per-language file sets merge into one system: text from
// every language, everything else from the first.
func TestFetch_Languages(t *testing.T) {
	en := httptest.NewServer(NewTestGbfsServer("en", testdata.Path("server/gbfs")))
	defer en.Close()
	listed := SystemFile{}
	if _, err := fetchUnmarshal(en.URL+"/gbfs.json", &listed, request.WithAllowHTTPUnfiltered); err != nil {
		t.Fatal(err)
	}

	// French and Dutch text as a 1.x/2.x file publishes it: plain strings. The
	// French system_information claims English, as Citi Bike's does. Neither
	// language's realtime files may be fetched, and a French file that fails
	// first in its list must not cost the rest.
	mux := http.NewServeMux()
	other := httptest.NewServer(mux)
	defer other.Close()
	files := func(lang string, served map[string]any) *SystemFeeds {
		ret := &SystemFeeds{}
		for _, name := range []string{fileSystemRegions, fileSystemInformation, fileStationInformation, fileStationStatus} {
			path := "/" + lang + "/" + name + ".json"
			ret.Feeds = append(ret.Feeds, &SystemFeed{Name: tt.NewString(name), URL: tt.NewString(other.URL + path)})
			if name == fileStationStatus {
				mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
					t.Errorf("fetched %s", path)
				})
			} else if v, ok := served[name]; ok {
				mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
					json.NewEncoder(w).Encode(v)
				})
			}
		}
		return ret
	}
	fr := files("fr", map[string]any{
		fileSystemInformation: map[string]any{"data": map[string]any{"language": "en", "name": "Vélos de la Baie"}},
		fileStationInformation: map[string]any{"data": map[string]any{"stations": []map[string]any{
			{"station_id": "68c89d1f-407a-4550-a2b7-ecf0ad7ee422", "name": "Rue San Carlos"},
			{"station_id": "fr-only", "name": "Nulle part"},
		}}},
	})
	nl := files("nl", map[string]any{
		fileSystemInformation: map[string]any{"data": map[string]any{"name": "Fietsen van de Baai"}},
	})

	// Listed in reverse: a small Go map iterates as a rotation of the order its
	// keys went in, so reversed keys never come out sorted by accident.
	discovery := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string][]byte{}
		for lang, sf := range map[string]*SystemFeeds{"nl": nl, "fr": fr, "en": listed.Data["en"]} {
			body[lang], _ = json.Marshal(sf)
		}
		fmt.Fprintf(w, `{"data":{"nl":%s,"fr":%s,"en":%s}}`, body["nl"], body["fr"], body["en"])
	}))
	defer discovery.Close()

	opts := Options{}
	opts.FeedURL = discovery.URL
	opts.AllowHTTPFetchUnfiltered = true
	feed, _, err := Fetch(context.Background(), nil, opts)
	if err != nil {
		t.Fatal(err)
	}
	if !assert.NotNil(t, feed) {
		return
	}
	assert.Equal(t, []string{"en", "fr", "nl"}, feed.SystemInformation.Languages.Val)
	assert.Equal(t, "en", feed.SystemInformation.Language.Val)
	assert.Equal(t, LocalizedString{
		{Text: "Bay Wheels", Language: "en"},
		{Text: "Vélos de la Baie", Language: "fr"},
		{Text: "Fietsen van de Baai", Language: "nl"},
	}, feed.SystemInformation.Name)
	stations := map[string]LocalizedString{}
	for _, s := range feed.StationInformation {
		stations[s.StationID.Val] = s.Name
	}
	assert.NotContains(t, stations, "fr-only")
	assert.Equal(t, LocalizedText{Text: "Rue San Carlos", Language: "fr"}, stations["68c89d1f-407a-4550-a2b7-ecf0ad7ee422"][1])
	assert.NotEmpty(t, feed.StationStatus)
}

// A language whose system_information fails is passed over. One whose other
// files fail keeps the system it did fetch, and a later language supplies the
// entities it could not.
func TestFetch_LanguageFallback(t *testing.T) {
	en := httptest.NewServer(NewTestGbfsServer("en", testdata.Path("server/gbfs")))
	defer en.Close()
	listed := SystemFile{}
	if _, err := fetchUnmarshal(en.URL+"/gbfs.json", &listed, request.WithAllowHTTPUnfiltered); err != nil {
		t.Fatal(err)
	}
	enURL := func(name string) string {
		for _, f := range listed.Data["en"].Feeds {
			if f.Name.Val == name {
				return f.URL.Val
			}
		}
		return ""
	}
	mux := http.NewServeMux()
	other := httptest.NewServer(mux)
	defer other.Close()
	mux.HandleFunc("/ca/system_information.json", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"name": "Bicis de la Badia"}})
	})
	feeds := func(urls map[string]string) *SystemFeeds {
		ret := &SystemFeeds{}
		for name, url := range urls {
			ret.Feeds = append(ret.Feeds, &SystemFeed{Name: tt.NewString(name), URL: tt.NewString(url)})
		}
		return ret
	}
	fetch := func(t *testing.T, data map[string]*SystemFeeds) *GbfsFeed {
		t.Helper()
		discovery := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(SystemFile{Data: data})
		}))
		defer discovery.Close()
		opts := Options{}
		opts.FeedURL = discovery.URL
		opts.AllowHTTPFetchUnfiltered = true
		feed, _, err := Fetch(context.Background(), nil, opts)
		if err != nil {
			t.Fatal(err)
		}
		if feed == nil {
			t.Fatal("no system")
		}
		return feed
	}

	t.Run("first language has no system_information", func(t *testing.T) {
		feed := fetch(t, map[string]*SystemFeeds{
			"ca": feeds(map[string]string{fileSystemInformation: other.URL + "/missing.json"}),
			"en": listed.Data["en"],
		})
		assert.Equal(t, []string{"en"}, feed.SystemInformation.Languages.Val)
		assert.NotEmpty(t, feed.StationInformation)
		assert.NotEmpty(t, feed.StationStatus)
	})

	t.Run("first language has no stations", func(t *testing.T) {
		feed := fetch(t, map[string]*SystemFeeds{
			"ca": feeds(map[string]string{
				fileSystemInformation:  other.URL + "/ca/system_information.json",
				fileStationInformation: other.URL + "/missing.json",
				fileStationStatus:      enURL(fileStationStatus),
			}),
			"en": listed.Data["en"],
		})
		assert.Equal(t, []string{"ca", "en"}, feed.SystemInformation.Languages.Val)
		if assert.NotEmpty(t, feed.StationInformation) {
			assert.Equal(t, "en", feed.StationInformation[0].Name[0].Language)
		}
		assert.NotEmpty(t, feed.StationStatus)
	})
}

// A 3.x feed is read as it is published: localized text, RFC3339 times and the
// renamed vehicle fields.
func TestFetch_V3(t *testing.T) {
	ts := httptest.NewServer(NewTestGbfsServer("", testdata.Path("server/gbfs-v3")))
	defer ts.Close()
	opts := Options{}
	opts.FeedURL = ts.URL + "/gbfs.json"
	opts.AllowHTTPFetchUnfiltered = true
	feed, _, err := Fetch(context.Background(), nil, opts)
	if err != nil {
		t.Fatal(err)
	}
	if !assert.NotNil(t, feed) {
		return
	}
	assert.Equal(t, "en", feed.SystemInformation.Language.Val)
	assert.Equal(t, LocalizedString{{Text: "Example Bike Rental", Language: "en"}, {Text: "Location de vélos", Language: "fr"}}, feed.SystemInformation.Name)
	if assert.Len(t, feed.StationInformation, 1) {
		assert.Equal(t, "Main Street", feed.StationInformation[0].Name.Default())
		assert.True(t, feed.StationInformation[0].ParkingHoop.Val)
		// A plain string where 3.x wants localized text takes the first language.
		assert.Equal(t, LocalizedString{{Text: "MS", Language: "en"}}, feed.StationInformation[0].ShortName)
	}
	if assert.Len(t, feed.StationStatus, 1) {
		s := feed.StationStatus[0]
		assert.Equal(t, int64(6), s.NumBikesAvailable.Val)
		assert.Equal(t, int64(1), s.NumBikesDisabled.Val)
		assert.Equal(t, int64(1689593653), s.LastReported.Val)
	}
	if assert.Len(t, feed.Bikes, 1) {
		b := feed.Bikes[0]
		assert.Equal(t, "973a5c94", b.BikeID.Val)
		assert.Equal(t, int64(1689593653), b.LastReported.Val)
		assert.Equal(t, int64(1689609600), b.AvailableUntil.Val)
	}
}
