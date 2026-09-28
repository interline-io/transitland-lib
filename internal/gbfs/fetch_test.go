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

// A 1.x/2.x feed lists its files once per language. They are fetched as one
// system, with each language's text, and the first language's everything else.
func TestFetch_Languages(t *testing.T) {
	en := httptest.NewServer(NewTestGbfsServer("en", testdata.Path("server/gbfs")))
	defer en.Close()
	listed := SystemFile{}
	if _, err := fetchUnmarshal(en.URL+"/gbfs.json", &listed, request.WithAllowHTTPUnfiltered); err != nil {
		t.Fatal(err)
	}

	// French text for the system and one station, and a station English lacks.
	// Its realtime files must not be fetched, and its missing regions file must
	// not cost the rest.
	mux := http.NewServeMux()
	fr := httptest.NewServer(mux)
	defer fr.Close()
	serve := func(path string, v any) {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			if v == nil {
				t.Errorf("fetched %s", path)
			}
			json.NewEncoder(w).Encode(v)
		})
	}
	si := SystemInformationFile{Data: &SystemInformation{Name: LocalizedString{{Text: "Vélos de la Baie"}}}}
	st := StationInformationFile{}
	st.Data.Stations = []*StationInformation{
		{StationID: tt.NewString("68c89d1f-407a-4550-a2b7-ecf0ad7ee422"), Name: LocalizedString{{Text: "Rue San Carlos"}}},
		{StationID: tt.NewString("fr-only"), Name: LocalizedString{{Text: "Nulle part"}}},
	}
	serve("/system_information.json", si)
	serve("/station_information.json", st)
	serve("/station_status.json", nil)
	frFeeds := &SystemFeeds{}
	for _, name := range []string{"system_information", "station_information", "station_status", "system_regions"} {
		frFeeds.Feeds = append(frFeeds.Feeds, &SystemFeed{Name: tt.NewString(name), URL: tt.NewString(fr.URL + "/" + name + ".json")})
	}
	discovery := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(SystemFile{Data: map[string]*SystemFeeds{"fr": frFeeds, "en": listed.Data["en"]}})
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
	assert.Equal(t, []string{"en", "fr"}, feed.SystemInformation.Languages.Val)
	assert.Equal(t, "en", feed.SystemInformation.Language.Val)
	assert.Equal(t, LocalizedString{{Text: "Bay Wheels", Language: "en"}, {Text: "Vélos de la Baie", Language: "fr"}}, feed.SystemInformation.Name)
	stations := map[string]LocalizedString{}
	for _, s := range feed.StationInformation {
		stations[s.StationID.Val] = s.Name
	}
	assert.NotContains(t, stations, "fr-only")
	assert.Equal(t, "fr", stations["68c89d1f-407a-4550-a2b7-ecf0ad7ee422"][1].Language)
	assert.Equal(t, "Rue San Carlos", stations["68c89d1f-407a-4550-a2b7-ecf0ad7ee422"][1].Text)
	assert.NotEmpty(t, feed.StationStatus)
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
