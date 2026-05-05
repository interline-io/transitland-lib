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
	"github.com/interline-io/transitland-lib/server/auth/fga"
	"github.com/interline-io/transitland-lib/server/dbutil"
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
	// AllowAll installs an AllowAllChecker. Required for tests that exercise
	// mutations. Ignored when FGAEndpoint is set.
	AllowAll bool
}

func Config(t testing.TB, opts Options) model.Config {
	ctx := context.Background()
	db := testutil.MustOpenTestDB(t)
	return newTestConfig(t, ctx, db, opts)
}

func ConfigTx(t testing.TB, opts Options, cb func(model.Config) error) {
	ctx := context.Background()
	// Start Txn
	db := testutil.MustOpenTestDB(t)
	tx := db.MustBeginTx(ctx, nil)
	defer tx.Rollback()

	// Get finders
	testEnv := newTestConfig(t, ctx, tx, opts)

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
	db = dbutil.WithQueryLogger(db, false, 0)

	// Default time
	if opts.WhenUtc == "" {
		opts.WhenUtc = "2022-09-01T00:00:00Z"
	}

	when, err := time.Parse("2006-01-02T15:04:05Z", opts.WhenUtc)
	if err != nil {
		t.Fatal(err)
	}
	cl := &clock.Mock{T: when}

	// model.Config requires a non-nil Checker. Default to deny-all; read-side
	// tests still see public feeds via IncludePublic=true below.
	var checker model.Checker = &authz.DenyAllChecker{}
	if opts.AllowAll {
		checker = &authz.AllowAllChecker{}
	}
	if opts.FGAEndpoint != "" {
		fgaClient, fgaErr := fga.NewFGAClient(opts.FGAEndpoint, "", "")
		if fgaErr != nil {
			t.Fatal(fgaErr)
		}
		if opts.FGAModelFile != "" {
			if _, fgaErr := fgaClient.CreateStore(ctx, "test"); fgaErr != nil {
				t.Fatal(fgaErr)
			}
			if _, fgaErr := fgaClient.CreateModel(ctx, opts.FGAModelFile); fgaErr != nil {
				t.Fatal(fgaErr)
			}
		}
		for _, tk := range opts.FGAModelTuples {
			ltk, _, lookupErr := azchecker.EKLookup(db, tk)
			if lookupErr != nil {
				t.Fatal(lookupErr)
			}
			if fgaErr := fgaClient.WriteTuple(ctx, ltk); fgaErr != nil {
				t.Fatal(fgaErr)
			}
		}
		checker = azchecker.NewCheckerFromConfig(azchecker.CheckerConfig{}, nil, fgaClient, db)
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
		Finder:                   dbf,
		RTFinder:                 rtf,
		GbfsFinder:               gbf,
		Checker:                  checker,
		IncludePublic:            true,
		JobQueue:                 jobQueue,
		Actions:                  actionFinder,
		Clock:                    cl,
		Storage:                  opts.Storage,
		RTStorage:                opts.RTStorage,
		MaxRadius:                100_000,
		AllowHTTPFetchUnfiltered: true,
	}
}
