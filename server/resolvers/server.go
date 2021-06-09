package resolvers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/interline-io/transitland-lib/server/auth"
	"github.com/interline-io/transitland-lib/server/find"
	generated "github.com/interline-io/transitland-lib/server/generated/gqlgen"
	"github.com/interline-io/transitland-lib/server/model"
)

func NewServer() http.Handler {
	c := generated.Config{Resolvers: &Resolver{}}
	c.Directives.HasRole = func(ctx context.Context, obj interface{}, next graphql.Resolver, role model.Role) (interface{}, error) {
		user := auth.ForContext(ctx)
		if user == nil {
			user = &auth.User{}
		}
		if !user.HasRole(role) {
			return nil, fmt.Errorf("Access denied")
		}
		return next(ctx)
	}
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(c))
	return find.Middleware(model.DB, srv)
}
