package gql

import (
	"net/http"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/interline-io/transitland-lib/internal/generated/gqlout"
	"github.com/vektah/gqlparser/v2/ast"
)

// devMaxUpload is what NewDefaultHandler allows, deliberately far above gqlgen's
// 32 MiB default: its callers are tests, the demo command and the wasm bridge
const devMaxUpload int64 = 512 << 20 // 512 MiB

// NewExecutableSchema returns the generated schema bound to the resolvers, for a
// caller building its own gqlgen server.
func NewExecutableSchema() graphql.ExecutableSchema {
	return gqlout.NewExecutableSchema(gqlout.Config{Resolvers: &Resolver{}})
}

// NewDefaultHandler builds a gqlgen handler over the schema, wrapped in
// LoaderMiddleware, for tests, the demo command and the wasm bridge.
func NewDefaultHandler() http.Handler {
	srv := handler.New(NewExecutableSchema())
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{
		MaxUploadSize: devMaxUpload,
		MaxMemory:     devMaxUpload,
	})
	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	srv.Use(extension.Introspection{})

	return LoaderMiddleware(srv)
}
