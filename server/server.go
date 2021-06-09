package server

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/interline-io/transitland-lib/server/auth"
	"github.com/interline-io/transitland-lib/server/config"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/resolvers"
	"github.com/interline-io/transitland-lib/server/rest"
	_ "github.com/lib/pq"
)

func Serve(cfg config.Config) error {
	// Open database
	model.DB = model.MustOpenDB(cfg.DBURL)

	// Setup CORS and logging
	root := mux.NewRouter()
	cors := handlers.CORS(
		handlers.AllowedHeaders([]string{"content-type", "apikey", "authorization"}),
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowCredentials(),
	)
	root.Use(cors)
	root.Use(loggingMiddleware)

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
	graphqlServer := resolvers.NewServer()
	mount(root, "/rest", rest.NewServer(cfg, graphqlServer))
	root.Handle("/query", graphqlServer).Methods(http.MethodGet, http.MethodPost, http.MethodOptions)
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

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.RequestURI)
		next.ServeHTTP(w, r)
	})
}
