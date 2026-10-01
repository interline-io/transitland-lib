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
		// An absent marker means the path was rewritten; the bare prefix is
		// incomplete, but splicing the request path in would be wrong.
		{"marker absent falls back to prefix", "https://transit.land/api/v2", "/rest/something-else", "/openapi.json", "https://transit.land/api/v2"},
		// Decoded paths can carry a second literal marker, so matching the last
		// one would anchor the mount inside caller-controlled input.
		{"marker recurring in a path parameter", "https://transit.land/api/v2", "/rest/onestop_id/f-/onestop_id/evil", "/onestop_id/", "https://transit.land/api/v2/rest"},
		// A relative result reading as protocol-relative would redirect off-site.
		// Backslashes count: browsers resolve /\evil.com like //evil.com.
		{"protocol-relative mount refused", "", "//evil.com/openapi.json", "/openapi.json", ""},
		{"protocol-relative mount refused, deep", "", "//evil.com/a/openapi.json", "/openapi.json", ""},
		{"backslash mount refused", "", `/\evil.com/openapi.json`, "/openapi.json", ""},
		{"mixed slash mount refused", "", `/\/evil.com/openapi.json`, "/openapi.json", ""},
		{"backslash mount refused, root redirect", "", `/\evil.com/`, "", ""},
		// An absolute prefix already fixes the authority, so the same mounts are
		// only path noise and are kept.
		{"absolute prefix keeps odd mount", "https://transit.land/api/v2", "//evil.com/openapi.json", "/openapi.json", "https://transit.land/api/v2//evil.com"},
		{"absolute prefix keeps backslash mount", "https://transit.land/api/v2", `/\evil.com/openapi.json`, "/openapi.json", `https://transit.land/api/v2/\evil.com`},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, mountPrefix(tc.restPrefix, tc.urlPath, tc.routeMarker))
		})
	}
}

// serveRest mounts h under mountPath with cfg on the context and returns the
// response for a GET of reqPath. mountPath may be empty to serve at the root.
func serveRest(t testing.TB, h http.Handler, mountPath string, cfg model.Config, reqPath string) *httptest.ResponseRecorder {
	t.Helper()
	root := chi.NewRouter()
	if mountPath == "" {
		root.Mount("/", h)
	} else {
		root.Mount(mountPath, h)
	}
	req := httptest.NewRequest(http.MethodGet, reqPath, nil)
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
	// Guards the bug where RestPrefix passed straight through drops the mount, so
	// a client resolves /stops to /api/v2/stops (404) not /api/v2/rest/stops.
	t.Run("mounted with absolute prefix includes mount segment", func(t *testing.T) {
		cfg := model.Config{RestPrefix: "https://example.test/api/v2"}
		rr := serveRest(t, NewOpenAPIHandler(), "/rest", cfg, "/rest/openapi.json")
		require.Equal(t, http.StatusOK, rr.Code)
		servers := openAPIServers(t, rr.Body.Bytes())
		require.Len(t, servers, 1)
		assert.Equal(t, "https://example.test/api/v2/rest", servers[0]["url"])
	})

	t.Run("unmounted with no prefix emits no servers block", func(t *testing.T) {
		rr := serveRest(t, NewOpenAPIHandler(), "", model.Config{}, "/openapi.json")
		require.Equal(t, http.StatusOK, rr.Code)
		assert.Empty(t, openAPIServers(t, rr.Body.Bytes()))
	})
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
			rr := serveRest(t, r, "/rest", cfg, "/rest/onestop_id/"+tc.onestopId)
			assert.Equal(t, http.StatusFound, rr.Code)
			assert.Equal(t, tc.expect, rr.Header().Get("Location"))
		})
	}

	t.Run("unrecognized prefix is a 404", func(t *testing.T) {
		rr := serveRest(t, r, "/rest", cfg, "/rest/onestop_id/x-nope")
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	// chi.URLParam stays encoded while r.URL.Path is decoded, so deriving the
	// mount from the parameter broke for any id containing ~ — which route
	// Onestop IDs routinely do.
	t.Run("percent-encoded onestop_id still resolves the mount", func(t *testing.T) {
		encoded := []struct {
			path   string
			expect string
		}{
			{"/rest/onestop_id/r-9q9-antioch%7Esfia", "https://example.test/api/v2/rest/routes/r-9q9-antioch%7Esfia"},
			{"/rest/onestop_id/f-9q9-bart%2Fx", "https://example.test/api/v2/rest/feeds/f-9q9-bart%2Fx"},
			// The decoded path here contains a second "/onestop_id/", which is
			// why the mount is recovered from the marker's first occurrence.
			{"/rest/onestop_id/f-%2Fonestop_id%2Fevil", "https://example.test/api/v2/rest/feeds/f-%2Fonestop_id%2Fevil"},
		}
		for _, tc := range encoded {
			rr := serveRest(t, r, "/rest", cfg, tc.path)
			assert.Equal(t, http.StatusFound, rr.Code)
			assert.Equal(t, tc.expect, rr.Header().Get("Location"))
		}
	})
}
