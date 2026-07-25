package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests do not touch the database: GenerateOpenAPI walks the compiled
// gqlgen schema, and the redirect handlers only read config off the context.

func TestMountPrefix(t *testing.T) {
	tcs := []struct {
		name        string
		restPrefix  string
		urlPath     string
		routeMarker string
		expect      string
	}{
		{"mounted, absolute prefix", "https://transit.land/api/v2", "/rest/openapi.json", "/openapi.json", "https://transit.land/api/v2/rest"},
		{"mounted, root redirect", "https://transit.land/api/v2", "/rest/", "", "https://transit.land/api/v2/rest"},
		{"mounted, no trailing slash", "https://transit.land/api/v2", "/rest", "", "https://transit.land/api/v2/rest"},
		{"mounted, onestop_id", "https://transit.land/api/v2", "/rest/onestop_id/f-9q9-bart", "/onestop_id/", "https://transit.land/api/v2/rest"},
		{"nested mount", "https://transit.land/api/v2", "/a/b/openapi.json", "/openapi.json", "https://transit.land/api/v2/a/b"},
		{"unmounted, absolute prefix", "https://transit.land/api/v2", "/openapi.json", "/openapi.json", "https://transit.land/api/v2"},
		{"no prefix, mounted", "", "/rest/openapi.json", "/openapi.json", "/rest"},
		{"no prefix, unmounted", "", "/openapi.json", "/openapi.json", ""},
		{"no prefix, root redirect", "", "/", "", ""},
		// A marker the path does not contain means something rewrote the path.
		// Falling back to the bare prefix is merely incomplete; splicing the
		// request path in would be actively wrong.
		{"marker absent falls back to prefix", "https://transit.land/api/v2", "/rest/something-else", "/openapi.json", "https://transit.land/api/v2"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, mountPrefix(tc.restPrefix, tc.urlPath, tc.routeMarker))
		})
	}
}

// serveRest mounts h under mountPath with cfg on the context and returns the
// response for a GET of reqPath. mountPath may be empty to serve at the root.
func serveRest(t testing.TB, h http.Handler, mountPath string, cfg model.Config, reqPath string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	root := chi.NewRouter()
	if mountPath == "" {
		root.Mount("/", h)
	} else {
		root.Mount(mountPath, h)
	}
	req := httptest.NewRequest(http.MethodGet, reqPath, nil)
	if header != nil {
		req.Header = header
	}
	req = req.WithContext(model.WithConfig(req.Context(), cfg))
	rr := httptest.NewRecorder()
	root.ServeHTTP(rr, req)
	return rr
}

func openAPIServers(t testing.TB, body []byte) []map[string]any {
	t.Helper()
	var doc struct {
		Servers []map[string]any `json:"servers"`
	}
	require.NoError(t, json.Unmarshal(body, &doc))
	return doc.Servers
}

func TestOpenAPIHandlerServers(t *testing.T) {
	// The bug this guards: passing RestPrefix straight through drops the mount
	// segment, so a generated client resolves /stops to
	// https://transit.land/api/v2/stops (404) instead of
	// https://transit.land/api/v2/rest/stops.
	t.Run("mounted with absolute prefix includes mount segment", func(t *testing.T) {
		cfg := model.Config{RestPrefix: "https://example.test/api/v2"}
		rr := serveRest(t, NewOpenAPIHandler(), "/rest", cfg, "/rest/openapi.json", nil)
		require.Equal(t, http.StatusOK, rr.Code)
		servers := openAPIServers(t, rr.Body.Bytes())
		require.Len(t, servers, 1)
		assert.Equal(t, "https://example.test/api/v2/rest", servers[0]["url"])
	})

	t.Run("unmounted with no prefix emits no servers block", func(t *testing.T) {
		rr := serveRest(t, NewOpenAPIHandler(), "", model.Config{}, "/openapi.json", nil)
		require.Equal(t, http.StatusOK, rr.Code)
		assert.Empty(t, openAPIServers(t, rr.Body.Bytes()))
	})
}

func TestOpenAPIHandlerCaching(t *testing.T) {
	cfg := model.Config{RestPrefix: "https://example.test/api/v2"}
	h := NewOpenAPIHandler()

	first := serveRest(t, h, "/rest", cfg, "/rest/openapi.json", nil)
	require.Equal(t, http.StatusOK, first.Code)
	etag := first.Header().Get("ETag")
	require.NotEmpty(t, etag)
	assert.Equal(t, "application/json", first.Header().Get("Content-Type"))
	assert.Equal(t, openAPICacheControl, first.Header().Get("Cache-Control"))

	t.Run("repeat request is byte-identical with a stable etag", func(t *testing.T) {
		second := serveRest(t, h, "/rest", cfg, "/rest/openapi.json", nil)
		require.Equal(t, http.StatusOK, second.Code)
		assert.Equal(t, etag, second.Header().Get("ETag"))
		assert.Equal(t, first.Body.Bytes(), second.Body.Bytes())
	})

	t.Run("If-None-Match revalidates to 304 with no body", func(t *testing.T) {
		hdr := http.Header{"If-None-Match": []string{etag}}
		rr := serveRest(t, h, "/rest", cfg, "/rest/openapi.json", hdr)
		assert.Equal(t, http.StatusNotModified, rr.Code)
		assert.Empty(t, rr.Body.Bytes())
	})

	t.Run("a stale etag still gets the document", func(t *testing.T) {
		hdr := http.Header{"If-None-Match": []string{`"stale"`}}
		rr := serveRest(t, h, "/rest", cfg, "/rest/openapi.json", hdr)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, first.Body.Bytes(), rr.Body.Bytes())
	})
}

func TestOpenAPICacheGeneratesOncePerServerURL(t *testing.T) {
	c := &openAPICache{docs: map[string]*openAPIDocument{}}

	first, err := c.get("https://example.test/api/v2/rest")
	require.NoError(t, err)
	second, err := c.get("https://example.test/api/v2/rest")
	require.NoError(t, err)
	// Same pointer means the second call was served from the cache rather than
	// re-walking every REST query against the GraphQL schema.
	assert.Same(t, first, second, "expected the cached document to be reused")

	other, err := c.get("https://saas.example.test/api/v2/rest")
	require.NoError(t, err)
	assert.NotSame(t, first, other, "a distinct server URL needs its own document")
	assert.NotEqual(t, first.etag, other.etag)
	assert.Equal(t, "https://saas.example.test/api/v2/rest", openAPIServers(t, other.body)[0]["url"])
}

func TestOnestopIdRedirectIncludesMountSegment(t *testing.T) {
	r := chi.NewRouter()
	r.Handle("/onestop_id/{onestop_id}", &OnestopIdEntityRedirectRequest{})

	tcs := []struct {
		onestopId string
		expect    string
	}{
		{"f-9q9-bayarearapidtransit", "https://example.test/api/v2/rest/feeds/f-9q9-bayarearapidtransit"},
		{"o-9q9-bart", "https://example.test/api/v2/rest/operators/o-9q9-bart"},
		{"s-9q9p1-embarcadero", "https://example.test/api/v2/rest/stops/s-9q9p1-embarcadero"},
		{"r-9q9-antioch~sfia", "https://example.test/api/v2/rest/routes/r-9q9-antioch~sfia"},
	}
	cfg := model.Config{RestPrefix: "https://example.test/api/v2"}
	for _, tc := range tcs {
		t.Run(tc.onestopId, func(t *testing.T) {
			rr := serveRest(t, r, "/rest", cfg, "/rest/onestop_id/"+tc.onestopId, nil)
			assert.Equal(t, http.StatusFound, rr.Code)
			assert.Equal(t, tc.expect, rr.Header().Get("Location"))
		})
	}

	t.Run("unrecognized prefix is a 404", func(t *testing.T) {
		rr := serveRest(t, r, "/rest", cfg, "/rest/onestop_id/x-nope", nil)
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	// chi.URLParam returns the raw percent-encoded value while r.URL.Path is
	// decoded. Deriving the mount from the parameter rather than the route's
	// literal segment produced a redirect containing the whole request path,
	// e.g. .../rest/onestop_id/r-9q9-antioch~sfia/routes/r-9q9-antioch%7Esfia.
	// Route Onestop IDs routinely contain ~, and clients do encode it.
	t.Run("percent-encoded onestop_id still resolves the mount", func(t *testing.T) {
		encoded := []struct {
			path   string
			expect string
		}{
			{"/rest/onestop_id/r-9q9-antioch%7Esfia", "https://example.test/api/v2/rest/routes/r-9q9-antioch%7Esfia"},
			{"/rest/onestop_id/f-9q9-bart%2Fx", "https://example.test/api/v2/rest/feeds/f-9q9-bart%2Fx"},
		}
		for _, tc := range encoded {
			rr := serveRest(t, r, "/rest", cfg, tc.path, nil)
			assert.Equal(t, http.StatusFound, rr.Code)
			assert.Equal(t, tc.expect, rr.Header().Get("Location"))
		}
	})
}
