package rest

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
	"github.com/stretchr/testify/assert"
)

// The hook exists so a host can re-add a parameter it took off the request
// before any handler saw it. These pin that it reaches the two URLs a client
// follows back to this server, and that it is not applied anywhere else.
func TestConfigLink(t *testing.T) {
	addKey := func(_ context.Context, u string) string {
		sep := "?"
		if strings.Contains(u, "?") {
			sep = "&"
		}
		return u + sep + "apikey=" + url.QueryEscape("secret")
	}

	t.Run("nil rewriter leaves the url alone", func(t *testing.T) {
		cfg := model.Config{}
		assert.Equal(t, "https://example.test/rest/feeds", cfg.Link(context.Background(), "https://example.test/rest/feeds"))
	})

	t.Run("rewriter is applied", func(t *testing.T) {
		cfg := model.Config{LinkURL: addKey}
		assert.Equal(t, "https://example.test/rest/feeds?apikey=secret",
			cfg.Link(context.Background(), "https://example.test/rest/feeds"))
	})

	t.Run("rewriter sees an existing query", func(t *testing.T) {
		cfg := model.Config{LinkURL: addKey}
		assert.Equal(t, "https://example.test/rest/feeds?after=10&apikey=secret",
			cfg.Link(context.Background(), "https://example.test/rest/feeds?after=10"))
	})
}
