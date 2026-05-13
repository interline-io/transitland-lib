package gbfs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/interline-io/transitland-lib/testdata"
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
	feeds, _, err := Fetch(context.Background(), nil, opts)
	if err != nil {
		t.Fatal(err)
	}
	fids := []string{}
	for _, ent := range feeds {
		fids = append(fids, ent.SystemInformation.Name.Val)
	}
	assert.ElementsMatch(t, []string{"Bay Wheels"}, fids)
}
