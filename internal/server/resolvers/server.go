package resolvers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/gorilla/mux"
	"github.com/interline-io/transitland-lib/internal/server/auth"
	"github.com/interline-io/transitland-lib/internal/server/config"
	"github.com/interline-io/transitland-lib/internal/server/find"
	generated "github.com/interline-io/transitland-lib/internal/server/generated/gqlgen"
	"github.com/interline-io/transitland-lib/internal/server/model"
)

func NewServer(cfg config.Config) (http.Handler, error) {
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

	// Setup server
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(c))
	graphqlServer := find.Middleware(model.DB, srv)
	root := mux.NewRouter()
	// Setup auth; default is all users will be anonymous.
	if cfg.UseAuth == "admin" {
		if m, err := auth.AdminAuthMiddleware(model.DB); err == nil {
			root.Use(m)
		} else {
			return nil, err
		}
	} else if cfg.UseAuth == "user" {
		if m, err := auth.UserAuthMiddleware(model.DB); err == nil {
			root.Use(m)
		} else {
			return nil, err
		}

	} else if cfg.UseAuth == "jwt" {
		if m, err := auth.JWTMiddleware(cfg); err == nil {
			root.Use(m)
		} else {
			return nil, err
		}
	}
	root.Handle("/", graphqlServer).Methods(http.MethodGet, http.MethodPost, http.MethodOptions)
	return root, nil
}
