package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/gql"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// linkMarker stands in for the parameter a host re-adds to links a caller
// follows without authenticating again.
const linkMarker = "testlink=1"

func appendMarker(_ context.Context, u string) string {
	if strings.Contains(u, "?") {
		return u + "&" + linkMarker
	}
	return u + "?" + linkMarker
}

func testHandlerWithLinkURL(t testing.TB, fn func(context.Context, string) string) http.Handler {
	t.Helper()
	cfg := testconfig.Config(t, testconfig.Options{Storage: testdata.Path("server", "tmp")})
	cfg.LinkURL = fn
	graphqlHandler, err := gql.NewServer()
	require.NoError(t, err)
	restHandler, err := NewServer(graphqlHandler)
	require.NoError(t, err)
	return model.AddConfigAndPerms(cfg, restHandler)
}

func TestConfigLink(t *testing.T) {
	t.Run("nil rewriter leaves the url alone", func(t *testing.T) {
		cfg := model.Config{}
		assert.Equal(t, "https://example.test/rest/feeds", cfg.Link(context.Background(), "https://example.test/rest/feeds"))
	})

	t.Run("rewriter is applied", func(t *testing.T) {
		cfg := model.Config{LinkURL: appendMarker}
		assert.Equal(t, "https://example.test/rest/feeds?"+linkMarker,
			cfg.Link(context.Background(), "https://example.test/rest/feeds"))
	})

	t.Run("rewriter sees an existing query", func(t *testing.T) {
		cfg := model.Config{LinkURL: appendMarker}
		assert.Equal(t, "https://example.test/rest/feeds?after=10&"+linkMarker,
			cfg.Link(context.Background(), "https://example.test/rest/feeds?after=10"))
	})
}

// The hook has to reach the URLs a client follows back into the API. Removing
// either cfg.Link call site must fail here.
func TestLinkURL_AppliedToFollowedLinks(t *testing.T) {
	srv := testHandlerWithLinkURL(t, appendMarker)

	t.Run("pagination link", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/feeds.json?limit=1", nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Result().StatusCode)

		var body struct {
			Meta struct {
				Next string `json:"next"`
			} `json:"meta"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		require.NotEmpty(t, body.Meta.Next, "expected a next link to assert on")
		assert.Contains(t, body.Meta.Next, linkMarker)
		// The rewrite must not cost the pagination cursor.
		assert.Contains(t, body.Meta.Next, "after=")
	})

	t.Run("onestop_id redirect", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/onestop_id/f-123", nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		require.Equal(t, http.StatusFound, w.Result().StatusCode)
		assert.Contains(t, w.Result().Header.Get("Location"), linkMarker)
	})
}

// The OpenAPI document serves anonymously, so a credential must not reach the
// redirect that points at it or the servers entry inside it.
func TestLinkURL_NotAppliedToPublicDocuments(t *testing.T) {
	srv := testHandlerWithLinkURL(t, appendMarker)

	t.Run("root redirect to openapi.json", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		assert.NotContains(t, w.Result().Header.Get("Location"), linkMarker)
	})

	t.Run("openapi servers entry", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/openapi.json", nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.NotContains(t, w.Body.String(), linkMarker)
	})
}
