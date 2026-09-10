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

// devMaxUpload allows uploads far larger than a deployment should accept, and
// devMaxMemory buffers them whole in memory rather than spilling to a temp file.
// Both are deliberate: this is what makes NewDefaultHandler unsuitable to serve,
// and the wasm bridge has no filesystem to spill to anyway.
//
// devMaxMemory must stay strictly above devMaxUpload — gqlgen buffers only when
// ContentLength is less than MaxMemory, so an upload of exactly devMaxUpload
// would otherwise take the temp-file path.
const (
	devMaxUpload int64 = 512 << 20 // 512 MiB
	devMaxMemory int64 = devMaxUpload + 1
)

// NewExecutableSchema returns the generated schema bound to the resolvers.
func NewExecutableSchema() graphql.ExecutableSchema {
	return gqlout.NewExecutableSchema(gqlout.Config{Resolvers: &Resolver{}})
}

// NewDefaultHandler builds a gqlgen handler over the schema, wrapped in
// LoaderMiddleware.
//
// It is deliberately not fit to serve: introspection is always on and uploads
// are accepted whole into memory with no meaningful cap. Anything serving real
// traffic builds its own from NewExecutableSchema and LoaderMiddleware.
func NewDefaultHandler() http.Handler {
	srv := handler.New(NewExecutableSchema())
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{
		MaxUploadSize: devMaxUpload,
		MaxMemory:     devMaxMemory,
	})
	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	srv.Use(extension.Introspection{})

	return LoaderMiddleware(srv)
}
