package gql

import (
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/interline-io/transitland-lib/internal/generated/gqlout"
)

func NewServer() (http.Handler, error) {
	c := gqlout.Config{Resolvers: &Resolver{}}
	// Setup server
	srv := handler.NewDefaultServer(gqlout.NewExecutableSchema(c))
	graphqlServer := loaderMiddleware(srv)
	return graphqlServer, nil
}
