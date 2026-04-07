package gql

import (
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/gorilla/websocket"
	"github.com/interline-io/transitland-lib/internal/generated/gqlout"
)

// ServerOption configures the gqlgen server instance
type ServerOption func(s *handler.Server)

// WithExtensions applies one or more gqlgen handler extensions to the server
func WithExtensions(exts ...graphql.HandlerExtension) ServerOption {
	return func(s *handler.Server) {
		for _, ext := range exts {
			if ext != nil {
				s.Use(ext)
			}
		}
	}
}

func NewServer(opts ...ServerOption) (http.Handler, error) {
	c := gqlout.Config{Resolvers: &Resolver{}}
	// Setup server with explicit transports to include WebSocket support
	srv := handler.New(gqlout.NewExecutableSchema(c))
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		InitTimeout:           30 * time.Second,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	})
	srv.Use(extension.Introspection{})
	// Apply functional options
	for _, opt := range opts {
		if opt != nil {
			opt(srv)
		}
	}
	graphqlServer := loaderMiddleware(srv)
	return graphqlServer, nil
}
