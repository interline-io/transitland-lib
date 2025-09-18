package gql

import (
	"net/http"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
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
	// Setup server
	srv := handler.NewDefaultServer(gqlout.NewExecutableSchema(c))
	// Apply functional options
	for _, opt := range opts {
		if opt != nil {
			opt(srv)
		}
	}
	graphqlServer := loaderMiddleware(srv)
	return graphqlServer, nil
}
