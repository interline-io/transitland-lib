package cmds

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"time"
	_ "time/tzdata"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/go-redis/redis/v8"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/auth/authz"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/meters"
	localmeter "github.com/interline-io/transitland-lib/server/meters/local"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tldb/querylogger"

	"github.com/interline-io/transitland-lib/server/finders/actions"
	"github.com/interline-io/transitland-lib/server/finders/dbfinder"
	"github.com/interline-io/transitland-lib/server/finders/gbfsfinder"
	"github.com/interline-io/transitland-lib/server/finders/rtfinder"
	"github.com/interline-io/transitland-lib/server/gql"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/playground"
	"github.com/interline-io/transitland-lib/server/rest"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"

	// Import drivers
	_ "github.com/interline-io/transitland-lib/tldb/postgres"

	// Import routers
	_ "github.com/interline-io/transitland-lib/server/directions/awsrouter"
	_ "github.com/interline-io/transitland-lib/server/directions/linerouter"
	_ "github.com/interline-io/transitland-lib/server/directions/tlrouter"
	_ "github.com/interline-io/transitland-lib/server/directions/valhalla"
)

type ServerCommand struct {
	Timeout                 int
	LongQueryDuration       int
	Port                    string
	RestPrefix              string
	LoadAdmins              bool
	ValidateLargeFiles      bool
	UseMaterialized         bool
	DisableAuth             bool
	LoaderBatchSize         int
	LoaderStopTimeBatchSize int
	SecretsFile             string
	Storage                 string
	RTStorage               string
	DBURL                   string
	RedisURL                string
	MaxRadius               float64
	secrets                 []dmfr.Secret
}

func (cmd *ServerCommand) HelpDesc() (string, string) {
	return "Run transitland server", ""
}

func (cmd *ServerCommand) HelpArgs() string {
	return "[flags]"
}

func (cmd *ServerCommand) AddFlags(fl *pflag.FlagSet) {
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	fl.StringVar(&cmd.RedisURL, "redisurl", "", "Redis URL (default: $TL_REDIS_URL)")
	fl.StringVar(&cmd.Storage, "storage", "", "Static storage backend")
	fl.StringVar(&cmd.RTStorage, "rt-storage", "", "RT storage backend")
	fl.BoolVar(&cmd.ValidateLargeFiles, "validate-large-files", false, "Allow validation of large files")
	fl.StringVar(&cmd.RestPrefix, "rest-prefix", "", "REST prefix for generating pagination links")
	fl.StringVar(&cmd.Port, "port", "8080", "")
	fl.StringVar(&cmd.SecretsFile, "secrets", "", "DMFR file containing secrets")
	fl.IntVar(&cmd.Timeout, "timeout", 60, "")
	fl.IntVar(&cmd.LongQueryDuration, "long-query", 1000, "Log queries over this duration (ms)")
	fl.BoolVar(&cmd.LoadAdmins, "load-admins", false, "Load admin polygons from database into memory")
	fl.IntVar(&cmd.LoaderBatchSize, "loader-batch-size", 100, "GraphQL Loader batch size")
	fl.IntVar(&cmd.LoaderStopTimeBatchSize, "loader-stop-time-batch-size", 1, "GraphQL Loader batch size for StopTimes")
	fl.Float64Var(&cmd.MaxRadius, "max-radius", 100_000, "Maximum radius for nearby stops")
	fl.BoolVar(&cmd.UseMaterialized, "use-materialized", false, "Use materialized views for active entities")
	fl.BoolVar(&cmd.DisableAuth, "disable-auth", false, "Disable feed authorization checks (treat all feeds as public)")
}

func (cmd *ServerCommand) Parse(args []string) error {
	// DB
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	if cmd.RedisURL == "" {
		cmd.RedisURL = os.Getenv("TL_REDIS_URL")
	}

	// Load secrets
	var secrets []dmfr.Secret
	if v := cmd.SecretsFile; v != "" {
		rr, err := dmfr.LoadAndParseRegistry(v)
		if err != nil {
			return errors.New("unable to load secrets file")
		}
		secrets = rr.Secrets
	}
	cmd.secrets = secrets
	return nil
}

func (cmd *ServerCommand) Run(ctx context.Context) error {
	// Open database
	var db tldb.Ext
	dbx, err := dbutil.OpenDB(cmd.DBURL)
	if err != nil {
		return err
	}
	db = dbx
	if log.Logger.GetLevel() == zerolog.TraceLevel {
		db = &querylogger.QueryLogger{Ext: dbx, Trace: true, LongQueryDuration: time.Duration(cmd.LongQueryDuration) * time.Millisecond}
	}

	// Open redis
	var redisClient *redis.Client
	if cmd.RedisURL != "" {
		redisClient, err = dbutil.OpenRedis(cmd.RedisURL)
		if err != nil {
			return err
		}
	}

	// Create Finder
	dbFinder := dbfinder.NewFinder(db)
	if cmd.LoadAdmins {
		dbFinder.LoadAdmins(ctx)
	}

	// Create RTFinder, GbfsFinder
	var rtFinder model.RTFinder
	var gbfsFinder model.GbfsFinder
	if redisClient != nil {
		// Use redis backed finders
		rtFinder = rtfinder.NewFinder(rtfinder.NewRedisCache(redisClient), db)
		gbfsFinder = gbfsfinder.NewFinder(redisClient)
	} else {
		// Default to in-memory cache
		rtFinder = rtfinder.NewFinder(rtfinder.NewLocalCache(), db)
		gbfsFinder = gbfsfinder.NewFinder(nil)
	}

	var actionFinder model.Actions = &actions.Actions{}

	// Setup config
	cfg := model.Config{
		Finder:                  dbFinder,
		RTFinder:                rtFinder,
		GbfsFinder:              gbfsFinder,
		Actions:                 actionFinder,
		Secrets:                 cmd.secrets,
		Storage:                 cmd.Storage,
		RTStorage:               cmd.RTStorage,
		ValidateLargeFiles:      cmd.ValidateLargeFiles,
		UseMaterialized:         cmd.UseMaterialized,
		RestPrefix:              cmd.RestPrefix,
		LoaderBatchSize:         cmd.LoaderBatchSize,
		LoaderStopTimeBatchSize: cmd.LoaderStopTimeBatchSize,
		MaxRadius:               cmd.MaxRadius,
	}

	// Disable auth if requested
	if cmd.DisableAuth {
		cfg.Checker = &globalAdminChecker{}
	}

	// Setup router
	root := chi.NewRouter()
	root.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"content-type", "apikey", "authorization"},
		AllowCredentials: true,
	}))

	// Finders config
	root.Use(model.AddConfig(cfg))

	// This server only supports admin access
	root.Use(usercheck.AdminDefaultMiddleware("admin"))

	// Add logging middleware - must be after auth
	root.Use(log.RequestIDMiddleware)
	root.Use(log.RequestIDLoggingMiddleware)
	root.Use(log.DurationLoggingMiddleware(cmd.LongQueryDuration, func(ctx context.Context) string {
		if user := authn.ForContext(ctx); user != nil {
			return user.Name()
		}
		return ""
	}))

	// PermFilter context
	root.Use(model.AddPerms(cfg.Checker))

	// Profiling
	root.HandleFunc("/debug/pprof/", pprof.Index)
	root.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	root.HandleFunc("/debug/pprof/profile", pprof.Profile)
	root.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

	// Metering and metrics
	meterProvider := localmeter.NewLocalMeterProvider()

	// GraphQL API
	graphqlServer, err := gql.NewServer()
	if err != nil {
		return err
	} else {
		r := chi.NewRouter()
		r.Use(meters.WithMeter(meterProvider, "graphql", 1.0, nil))
		r.Mount("/", graphqlServer)
		root.Mount("/query", r)
	}

	// REST API
	restServer, err := rest.NewServer(graphqlServer)
	if err != nil {
		return err
	} else {
		r := chi.NewRouter()
		r.Use(meters.WithMeter(meterProvider, "rest", 1.0, nil))
		r.Mount("/", restServer)
		root.Mount("/rest", r)
	}

	// GraphQL Playground
	root.Handle("/", playground.Handler("GraphQL playground", "/query"))

	// Start server
	timeOut := time.Duration(cmd.Timeout) * time.Second
	addr := fmt.Sprintf("%s:%s", "0.0.0.0", cmd.Port)
	log.For(ctx).Info().Msgf("Listening on: %s", addr)
	srv := &http.Server{
		Handler:      http.TimeoutHandler(root, timeOut, "timeout"),
		Addr:         addr,
		WriteTimeout: 2 * timeOut,
		ReadTimeout:  2 * timeOut,
	}
	return srv.ListenAndServe()
}

// globalAdminChecker is a simple checker that always returns true for CheckGlobalAdmin,
// effectively disabling all feed authorization checks.
type globalAdminChecker struct {
	authz.UnimplementedCheckerServer
}

func (c *globalAdminChecker) CheckGlobalAdmin(ctx context.Context) (bool, error) {
	return true, nil
}
