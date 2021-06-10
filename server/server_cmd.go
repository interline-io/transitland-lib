package server

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
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
)

type Command struct {
	config.Config
}

func (cmd *Command) Parse(args []string) error {
	fl := flag.NewFlagSet("sync", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: server")
		fl.PrintDefaults()
	}
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	fl.IntVar(&cmd.Timeout, "timeout", 60, "")
	fl.StringVar(&cmd.Port, "port", "8080", "")
	fl.StringVar(&cmd.JwtAudience, "jwt-audience", "", "JWT Audience")
	fl.StringVar(&cmd.JwtIssuer, "jwt-issuer", "", "JWT Issuer")
	fl.StringVar(&cmd.JwtPublicKeyFile, "jwt-public-key-file", "", "Path to JWT public key file")
	fl.StringVar(&cmd.UseAuth, "auth", "", "")
	fl.StringVar(&cmd.GtfsDir, "gtfsdir", "", "Directory to store GTFS files")
	fl.StringVar(&cmd.GtfsS3Bucket, "s3", "", "S3 bucket for GTFS files")
	fl.BoolVar(&cmd.ValidateLargeFiles, "validate-large-files", false, "Allow validation of large files")
	fl.Parse(args)
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	return nil
}

func (cmd *Command) Run(args []string) error {
	// TODO: fix interface
	if err := cmd.Parse(args); err != nil {
		panic(err)
	}
	return Serve(cmd.Config)
}

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
