package cmds

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/fetch"
	"github.com/interline-io/transitland-lib/stats"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/validator"
	"github.com/spf13/pflag"
)

type FetchCommandResult struct {
	Result                     fetch.Result
	FeedVersion                *dmfr.FeedVersion
	FeedVersionValidatorResult *validator.Result
	FatalError                 error
}

// FetchJob specifies a single feed fetch operation.
// This allows programmatic specification of multiple fetches with different URLs.
type FetchJob struct {
	FeedID  string // Feed identifier (onestop_id)
	FeedURL string // URL to fetch from (optional; if empty, looks up from database)
}

// FetchCommand fetches feeds defined a DMFR database.
type FetchCommand struct {
	Options     fetch.StaticFetchOptions
	SecretsFile string
	SecretEnv   []string
	CreateFeed  bool
	Workers     int
	Fail        bool
	Limit       int
	DBURL       string
	DryRun      bool
	FetchJobs   []FetchJob // List of fetch operations
	Results     []FetchCommandResult
	Adapter     tldb.Adapter // allow for mocks
	fetchedAt   string
	jobsFile    string
	dmfrFile    string
}

func (cmd *FetchCommand) HelpDesc() (string, string) {
	return "Fetch GTFS data and create feed versions", "Use after the `sync` command."
}

func (cmd *FetchCommand) HelpArgs() string {
	return "[flags] [feeds...]"
}

func (cmd *FetchCommand) AddFlags(fl *pflag.FlagSet) {
	// FetchCommand options
	fl.BoolVar(&cmd.CreateFeed, "create-feed", false, "Create feed record if not found")
	fl.BoolVar(&cmd.DryRun, "dry-run", false, "Dry run; print feeds that would be imported and exit")
	fl.BoolVar(&cmd.Fail, "fail", false, "Exit with error code if any fetch is not successful")
	fl.IntVar(&cmd.Limit, "limit", 0, "Maximum number of feeds to fetch")
	fl.IntVar(&cmd.Workers, "workers", 1, "Worker threads")
	fl.StringArrayVar(&cmd.SecretEnv, "secret-env", nil, "Specify secret from environment variable as feed_id:ENV_VAR or file.json:ENV_VAR")
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	fl.StringVar(&cmd.dmfrFile, "dmfr", "", "Filter by feed IDs in DMFR file; equivalent to specifying feed IDs as arguments")
	fl.StringVar(&cmd.fetchedAt, "fetched-at", "", "Manually specify fetched_at value, e.g. 2020-02-06T12:34:56Z")
	fl.StringVar(&cmd.jobsFile, "jobs-file", "", "Specify fetch jobs in file, one per line as 'feed_id <tab> url'")
	fl.StringVar(&cmd.SecretsFile, "secrets", "", "Path to DMFR Secrets file")
	// StaticFetchOptions
	fl.BoolVar(&cmd.Options.AllowFTPFetch, "allow-ftp-fetch", false, "Allow fetching from FTP urls")
	fl.BoolVar(&cmd.Options.AllowLocalFetch, "allow-local-fetch", false, "Allow fetching from filesystem directories/zip files")
	fl.BoolVar(&cmd.Options.AllowS3Fetch, "allow-s3-fetch", false, "Allow fetching from S3 urls")
	fl.BoolVar(&cmd.Options.SaveValidationReport, "validation-report", false, "Save validation report")
	fl.BoolVar(&cmd.Options.StrictValidation, "strict", false, "Reject feeds with validation errors")
	fl.StringVar(&cmd.Options.FeedURL, "feed-url", "", "Manually fetch a single URL; you must specify exactly one feed_id")
	fl.StringVar(&cmd.Options.Storage, "storage", ".", "Storage destination; can be s3://... az://... or path to a directory")
	fl.StringVar(&cmd.Options.ValidationReportStorage, "validation-report-storage", "", "Storage path for saving validation report JSON")
}

func (cmd *FetchCommand) Parse(args []string) error {
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	if cmd.Options.FeedURL != "" && len(args) != 1 {
		return errors.New("you must specify exactly one feed_id when using -feed-url")
	}
	for _, feedID := range args {
		cmd.FetchJobs = append(cmd.FetchJobs, FetchJob{FeedID: feedID, FeedURL: cmd.Options.FeedURL})
	}
	// Load feed IDs from DMFR file
	if cmd.dmfrFile != "" {
		reg, err := dmfr.LoadAndParseRegistry(cmd.dmfrFile)
		if err != nil {
			return err
		}
		for _, feed := range reg.Feeds {
			cmd.FetchJobs = append(cmd.FetchJobs, FetchJob{FeedID: feed.FeedID})
		}
	}
	// Read jobs file: each line is "feed_id<tab>url"
	if cmd.jobsFile != "" {
		lines, err := tlcli.ReadFileLines(cmd.jobsFile)
		if err != nil {
			return err
		}
		for _, line := range lines {
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) == 0 || parts[0] == "" {
				continue
			}
			job := FetchJob{FeedID: parts[0]}
			if len(parts) > 1 {
				job.FeedURL = parts[1]
			}
			cmd.FetchJobs = append(cmd.FetchJobs, job)
		}
	}
	return nil
}

// Run executes this command.
func (cmd *FetchCommand) Run(ctx context.Context) error {
	// Init
	if cmd.Workers < 1 {
		cmd.Workers = 1
	}
	if cmd.fetchedAt != "" {
		t, err := time.Parse(time.RFC3339Nano, cmd.fetchedAt)
		if err != nil {
			return err
		}
		cmd.Options.FetchedAt = t
	}
	if cmd.SecretsFile != "" {
		r, err := dmfr.LoadAndParseRegistry(cmd.SecretsFile)
		if err != nil {
			return err
		}
		cmd.Options.Secrets = r.Secrets
	}
	// Parse --secret-env arguments
	for _, se := range cmd.SecretEnv {
		secret, err := parseSecretEnv(se)
		if err != nil {
			return err
		}
		cmd.Options.Secrets = append(cmd.Options.Secrets, secret)
	}
	if cmd.Options.FetchedAt.IsZero() {
		cmd.Options.FetchedAt = time.Now()
	}
	cmd.Options.FetchedAt = cmd.Options.FetchedAt.UTC()

	// Get adapter
	if cmd.Adapter == nil {
		writer, err := tldb.OpenWriter(cmd.DBURL, true)
		if err != nil {
			return err
		}
		cmd.Adapter = writer.Adapter
		defer writer.Close()
	}
	adapter := cmd.Adapter

	// Populate FetchJobs from database if none specified
	if len(cmd.FetchJobs) == 0 {
		q := adapter.Sqrl().
			Select("*").
			From("current_feeds").
			Where("deleted_at IS NULL").
			Where("spec = ?", "gtfs")
		qstr, qargs, err := q.ToSql()
		if err != nil {
			return err
		}
		var feeds []dmfr.Feed
		if err = adapter.Select(ctx, &feeds, qstr, qargs...); err != nil {
			return err
		}
		for _, feed := range feeds {
			if feed.URLs.StaticCurrent == "" {
				continue
			}
			cmd.FetchJobs = append(cmd.FetchJobs, FetchJob{
				FeedID:  feed.FeedID,
				FeedURL: feed.URLs.StaticCurrent,
			})
		}
	}

	// Convert FetchJobs to internal fetchJob queue with full options
	// Apply limit if specified
	fetchJobs := cmd.FetchJobs
	if cmd.Limit > 0 && len(fetchJobs) > cmd.Limit {
		fetchJobs = fetchJobs[:cmd.Limit]
	}
	var toFetch []fetchJob
	for _, job := range fetchJobs {
		// Get feed, create if not present and CreateFeed is specified
		var feed dmfr.Feed
		if err := adapter.Get(ctx, &feed, `SELECT * FROM current_feeds WHERE onestop_id = ?`, job.FeedID); err == sql.ErrNoRows && cmd.CreateFeed {
			feed.FeedID = job.FeedID
			feed.Spec = "gtfs"
			if feed.ID, err = adapter.Insert(ctx, &feed); err != nil {
				return err
			}
		} else if err != nil {
			return fmt.Errorf("problem with feed '%s': %s", job.FeedID, err.Error())
		}
		// Create feed state if not exists
		if _, err := stats.EnsureFeedState(ctx, adapter, feed.ID); err != nil {
			return err
		}
		// Prepare options for this fetch
		opts := cmd.Options // copy global options
		opts.FeedID = feed.ID
		if job.FeedURL != "" {
			opts.URLType = "manual"
			opts.FeedURL = job.FeedURL
		} else {
			opts.URLType = "static_current"
			opts.FeedURL = feed.URLs.StaticCurrent
		}
		toFetch = append(toFetch, fetchJob{OnestopID: feed.FeedID, StaticFetchOptions: opts})
	}

	///////////////
	// Here we go
	log.For(ctx).Info().Msgf("Fetching %d feeds", len(cmd.FetchJobs))
	jobs := make(chan fetchJob, len(cmd.FetchJobs))
	results := make(chan FetchCommandResult, len(cmd.FetchJobs))
	for _, opts := range toFetch {
		jobs <- opts
	}
	close(jobs)

	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < cmd.Workers; w++ {
		wg.Add(1)
		go fetchWorker(ctx, cmd.Adapter, cmd.DryRun, jobs, results, &wg)
	}
	wg.Wait()
	close(results)

	// Check results
	var fatalError error
	fetchNew := 0
	fetchFound := 0
	fetchErrs := 0
	for result := range results {
		cmd.Results = append(cmd.Results, result)
		if result.FatalError != nil {
			// A fatal error occurred, always exit with error
			fatalError = result.FatalError
		} else if result.Result.FetchError != nil {
			fetchErrs++
			if cmd.Fail {
				// Exit with error if any fetch is not successful
				fatalError = result.Result.FetchError
			}
		} else if result.Result.Found {
			fetchFound++
		} else {
			fetchNew++
		}
	}
	log.For(ctx).Info().Msgf("Existing: %d New: %d Errors: %d", fetchFound, fetchNew, fetchErrs)
	if fatalError != nil {
		log.For(ctx).Info().Msgf("Exiting with error because at least one fetch had fatal error: %s", fatalError.Error())
		return fatalError
	}
	return nil
}

type fetchJob struct {
	OnestopID string
	fetch.StaticFetchOptions
}

func fetchWorker(ctx context.Context, adapter tldb.Adapter, DryRun bool, jobs <-chan fetchJob, results chan<- FetchCommandResult, wg *sync.WaitGroup) {
	for job := range jobs {
		jobLog := log.For(ctx).With().
			Str("feed_onestop_id", job.OnestopID).
			Int("feed_id", job.Options.FeedID).
			Str("url", job.FeedURL).
			Logger()

		jobLog.Info().Msg("start")
		if DryRun {
			jobLog.Info().Msg("dry-run")
			continue
		}

		// Fetch
		jobCtx := log.WithLogger(ctx, jobLog)
		var result fetch.StaticFetchResult
		t := time.Now()
		fatalError := adapter.Tx(func(atx tldb.Adapter) error {
			var fatalError error
			result, fatalError = fetch.StaticFetch(jobCtx, atx, job.StaticFetchOptions)
			return fatalError
		})
		t2 := float64(time.Now().UnixNano()-t.UnixNano()) / 1e9 // 1000000000.0

		// Log result
		fv := result.FeedVersion
		if fatalError != nil {
			jobLog.Error().Err(fatalError).Float64("duration", t2).Msg("critical error")
		} else if result.FetchError != nil {
			jobLog.Error().Err(result.FetchError).Float64("duration", t2).Msg("fetch error")
		} else if fv != nil && result.Found {
			jobLog.Info().
				Float64("duration", t2).
				Str("sha1", fv.SHA1).
				Int("feed_version_id", fv.ID).
				Msg("found existing")
		} else if fv != nil {
			jobLog.Info().
				Float64("duration", t2).
				Str("sha1", fv.SHA1).
				Int("feed_version_id", fv.ID).
				Msg("new feed version")
		} else {
			jobLog.Error().Float64("duration", t2).Msg("invalid result")
		}
		results <- FetchCommandResult{
			Result:                     result.Result,
			FeedVersion:                result.FeedVersion,
			FeedVersionValidatorResult: result.FeedVersionValidatorResult,
			FatalError:                 fatalError,
		}
	}
	wg.Done()
}

// parseSecretEnv parses a secret-env argument in the format "target:ENV_VAR"
// where target is either a feed_id or a filename (detected by .json suffix).
// The secret key value is read from the environment variable.
func parseSecretEnv(arg string) (dmfr.Secret, error) {
	parts := strings.SplitN(arg, ":", 2)
	if len(parts) != 2 {
		return dmfr.Secret{}, fmt.Errorf("invalid --secret-env format %q: expected target:ENV_VAR", arg)
	}
	target := parts[0]
	envVar := parts[1]
	if target == "" || envVar == "" {
		return dmfr.Secret{}, fmt.Errorf("invalid --secret-env format %q: target and ENV_VAR must not be empty", arg)
	}
	key := os.Getenv(envVar)
	if key == "" {
		return dmfr.Secret{}, fmt.Errorf("environment variable %q is not set or empty", envVar)
	}
	secret := dmfr.Secret{Key: key}
	if strings.HasSuffix(target, ".json") {
		secret.Filename = target
	} else {
		secret.FeedID = target
	}
	return secret, nil
}
