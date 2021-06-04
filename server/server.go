package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/interline-io/transitland-lib/server/auth"
	"github.com/interline-io/transitland-lib/server/config"
	"github.com/interline-io/transitland-lib/server/find"
	generated "github.com/interline-io/transitland-lib/server/generated/gqlgen"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/resolvers"
	"github.com/interline-io/transitland-lib/server/rest"
	_ "github.com/lib/pq"
)

func Serve(cfg config.Config) error {
	// Open database
	model.DB = model.MustOpenDB(cfg.DBURL)

	// Setup CORS
	root := mux.NewRouter()
	cors := handlers.CORS(
		handlers.AllowedHeaders([]string{"content-type", "apikey", "authorization"}),
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowCredentials(),
	)
	root.Use(cors)

	// Setup auth; default is all users will be anonymous.
	if cfg.UseAuth == "admin" {
		if m, err := auth.NoAuthMiddleware(model.DB); err == nil {
			root.Use(m)
		} else {
			return err
		}
	} else if cfg.UseAuth == "jwt" {
		if m, err := auth.JWTMiddleware(cfg); err == nil {
			root.Use(m)
		} else {
			return err
		}
	}

	// Add paths
	mount(root, "/rest", rest.MakeHandlers(cfg))
	root.Handle("/query", newServer()).Methods(http.MethodGet, http.MethodPost, http.MethodOptions)
	root.Handle("/", playground.Handler("GraphQL playground", "/query"))

	addr := fmt.Sprintf("%s:%s", "0.0.0.0", cfg.Port)
	fmt.Println("listening on:", addr)
	timeOut := time.Duration(cfg.Timeout)
	srv := &http.Server{
		Handler:      root,
		Addr:         addr,
		WriteTimeout: timeOut * time.Second,
		ReadTimeout:  timeOut * time.Second,
	}
	return srv.ListenAndServe()
}

func mount(r *mux.Router, path string, handler http.Handler) {
	r.PathPrefix(path).Handler(
		http.StripPrefix(
			strings.TrimSuffix(path, "/"),
			handler,
		),
	)
}

func newServer() http.Handler {
	c := generated.Config{Resolvers: &resolvers.Resolver{}}
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
