package gql

import (
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/interline-io/transitland-lib/internal/generated/gqlout"
	"github.com/vektah/gqlparser/v2/ast"
)

// DefaultMaxUploadSize caps the size of a multipart upload body (e.g.
// validate_gtfs / feed_version_fetch). This matches gqlgen's own conservative
// default; deployments that accept larger GTFS feeds should raise it with
// WithMaxUploadSize.
const DefaultMaxUploadSize int64 = 32 << 20 // 32 MiB

// maxMultipartMemory is the per-part threshold above which gqlgen spills an
// upload to a temp file instead of buffering it in memory. This is not an
// upload cap (that is MaxUploadSize); keep it modest.
const maxMultipartMemory int64 = 32 << 20 // 32 MiB

// serverConfig holds NewServer settings populated by ServerOptions before the
// gqlgen server is constructed.
type serverConfig struct {
	maxUploadSize      int64
	maxMultipartMemory int64
	extensions         []graphql.HandlerExtension
}

// ServerOption configures the gqlgen server instance
type ServerOption func(c *serverConfig)

// WithExtensions applies one or more gqlgen handler extensions to the server
func WithExtensions(exts ...graphql.HandlerExtension) ServerOption {
	return func(c *serverConfig) {
		c.extensions = append(c.extensions, exts...)
	}
}

// WithMaxUploadSize sets the maximum size, in bytes, of a multipart upload
// request body. Defaults to DefaultMaxUploadSize.
func WithMaxUploadSize(n int64) ServerOption {
	return func(c *serverConfig) {
		c.maxUploadSize = n
	}
}

// WithMaxMultipartMemory sets the per-part threshold above which gqlgen spills an
// upload to a temp file. Environments without a usable filesystem (js/wasm) must
// raise this to at least the upload size so nothing spills to disk.
func WithMaxMultipartMemory(n int64) ServerOption {
	return func(c *serverConfig) {
		c.maxMultipartMemory = n
	}
}

func NewServer(opts ...ServerOption) (http.Handler, error) {
	cfg := serverConfig{
		maxUploadSize:      DefaultMaxUploadSize,
		maxMultipartMemory: maxMultipartMemory,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	c := gqlout.Config{Resolvers: &Resolver{}}

	// Equivalent to handler.NewDefaultServer, but with a configurable multipart
	// upload cap. (gqlgen's default MultipartForm rejects bodies over 32 MiB,
	// and because it matches the first transport that supports a request, the
	// cap cannot be raised by adding a second MultipartForm after the fact — so
	// the server has to be built explicitly here.)
	srv := handler.New(gqlout.NewExecutableSchema(c))
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
	})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{
		MaxUploadSize: cfg.maxUploadSize,
		MaxMemory:     cfg.maxMultipartMemory,
	})
	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	// Apply caller-provided handler extensions.
	for _, ext := range cfg.extensions {
		if ext != nil {
			srv.Use(ext)
		}
	}

	graphqlServer := loaderMiddleware(srv)
	return graphqlServer, nil
}
