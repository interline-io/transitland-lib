package testconfig

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/internal/clock"
	"github.com/interline-io/transitland-lib/rt"
	"github.com/interline-io/transitland-lib/server/auth/authz"
	"github.com/interline-io/transitland-lib/server/auth/azchecker"
	"github.com/interline-io/transitland-lib/server/finders/actions"
	"github.com/interline-io/transitland-lib/server/finders/dbfinder"
	"github.com/interline-io/transitland-lib/server/finders/gbfsfinder"
	"github.com/interline-io/transitland-lib/server/finders/rtfinder"
	"github.com/interline-io/transitland-lib/server/jobs"
	localjobs "github.com/interline-io/transitland-lib/server/jobs/local"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/testdata"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tldb/querylogger"
	"google.golang.org/protobuf/proto"
)

// Test helpers

type Options struct {
	WhenUtc        string
	Storage        string
	RTStorage      string
	RTJsons        []RTJsonFile
	FGAEndpoint    string
	FGAModelFile   string
	FGAModelTuples []authz.TupleKey
}

func Config(t testing.TB, opts Options) model.Config {
	ctx := context.Background()
	db := testutil.MustOpenTestDB(t)
	return newTestConfig(t, ctx, &querylogger.QueryLogger{Ext: db}, opts)
}

func ConfigTx(t testing.TB, opts Options, cb func(model.Config) error) {
	ctx := context.Background()
	// Start Txn
	db := testutil.MustOpenTestDB(t)
	tx := db.MustBeginTx(ctx, nil)
	defer tx.Rollback()

	// Get finders
	testEnv := newTestConfig(t, ctx, &querylogger.QueryLogger{Ext: tx}, opts)

	// Commit or rollback
	if err := cb(testEnv); err != nil {
		//tx.Rollback()
	} else {
		tx.Commit()
	}
}

func ConfigTxRollback(t testing.TB, opts Options, cb func(model.Config)) {
	ConfigTx(t, opts, func(c model.Config) error {
		cb(c)
		return errors.New("rollback")
	})
}

type RTJsonFile struct {
	Feed  string
	Ftype string
	Fname string
}

func DefaultRTJson() []RTJsonFile {
	return []RTJsonFile{
		{"BA", "realtime_trip_updates", "BA.json"},
		{"BA", "realtime_alerts", "BA-alerts.json"},
		{"CT", "realtime_trip_updates", "CT.json"},
	}
}

func newTestConfig(t testing.TB, ctx context.Context, db tldb.Ext, opts Options) model.Config {
	// Default time
	if opts.WhenUtc == "" {
		opts.WhenUtc = "2022-09-01T00:00:00Z"
	}

	when, err := time.Parse("2006-01-02T15:04:05Z", opts.WhenUtc)
	if err != nil {
		t.Fatal(err)
	}
	cl := &clock.Mock{T: when}

	// Setup Checker
	var checker model.Checker
	if opts.FGAEndpoint != "" {
		checkerCfg := azchecker.CheckerConfig{
			FGAEndpoint:      opts.FGAEndpoint,
			FGALoadModelFile: opts.FGAModelFile,
			FGALoadTestData:  opts.FGAModelTuples,
		}
		checker, err = azchecker.NewCheckerFromConfig(ctx, checkerCfg, db)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Setup DB
	dbf := dbfinder.NewFinder(db)
	dbf.Clock = cl

	// Setup RT
	rtf := rtfinder.NewFinder(rtfinder.NewLocalCache(), db)
	rtf.Clock = cl
	for _, rtj := range opts.RTJsons {
		fn := testdata.Path("server", "rt", rtj.Fname)
		msg, err := rt.ReadFile(fn)
		if err != nil {
			t.Fatal(err)
		}
		key := fmt.Sprintf("rtdata:%s:%s", rtj.Feed, rtj.Ftype)
		rtdata, err := proto.Marshal(msg)
		if err != nil {
			t.Fatal(err)
		}
		if err := rtf.AddData(ctx, key, rtdata); err != nil {
			t.Fatal(err)
		}
	}

	// Setup GBFS
	gbf := gbfsfinder.NewFinder(nil)

	if opts.Storage == "" {
		opts.Storage = t.TempDir()
	}
	if opts.RTStorage == "" {
		opts.RTStorage = t.TempDir()
	}

	// Initialize job queue - do not start
	jobQueue := jobs.NewJobLogger(localjobs.NewLocalJobs())

	// Action finder
	actionFinder := &actions.Actions{}

	return model.Config{
		Finder:     dbf,
		RTFinder:   rtf,
		GbfsFinder: gbf,
		Checker:    checker,
		JobQueue:   jobQueue,
		Actions:    actionFinder,
		Clock:      cl,
		Storage:    opts.Storage,
		RTStorage:  opts.RTStorage,
		MaxRadius:  100_000,
	}
}
