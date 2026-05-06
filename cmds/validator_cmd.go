package cmds

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/internal/snakejson"
	"github.com/interline-io/transitland-lib/request"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/validator"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
)

// ValidatorCommand
type ValidatorCommand struct {
	Options                 validator.Options
	rtFiles                 []string
	OutputFile              string
	Quiet                   bool
	DBURL                   string
	FVID                    int
	extensionDefs           []string
	SaveValidationReport    bool
	ValidationReportStorage string
	readerPath              string
	errorThresholds         []string
	SecretsFile             string
	SecretEnv               []string
	DMFRFile                string
	FeedID                  string
	URLType                 string
	AllowFTPFetch           bool
	AllowLocalFetch         bool
	AllowS3Fetch            bool
	AllowHTTPFetchUnfiltered bool
	secrets                 []dmfr.Secret
}

func (cmd *ValidatorCommand) HelpDesc() (string, string) {
	return "Validate a GTFS feed", "The validate command performs a basic validation on a data source and writes the results to standard out."
}

func (cmd *ValidatorCommand) HelpExample() string {
	return `% {{.ParentCommand}} {{.Command}} "https://www.bart.gov/dev/schedules/google_transit.zip"
% {{.ParentCommand}} {{.Command}} -o - --include-entities "http://developer.trimet.org/schedule/gtfs.zip"
% {{.ParentCommand}} {{.Command}} --dmfr feeds/wmata.com.dmfr.json --feed-id f-dqc-wmata~rail --secrets secrets.json`
}

func (cmd *ValidatorCommand) HelpArgs() string {
	return "[flags] [<reader>]"
}

// shouldShowLogs returns true if logs should be displayed
func (cmd *ValidatorCommand) shouldShowLogs() bool {
	return !cmd.Quiet
}

func (cmd *ValidatorCommand) AddFlags(fl *pflag.FlagSet) {
	fl.StringSliceVar(&cmd.extensionDefs, "ext", nil, "Include GTFS Extension")
	fl.StringVarP(&cmd.OutputFile, "out", "o", "", "Write validation report as JSON to file; use '-' for stdout (implies -q)")
	fl.BoolVarP(&cmd.Quiet, "quiet", "q", false, "Suppress log output")
	fl.BoolVar(&cmd.Options.BestPractices, "best-practices", false, "Include Best Practices validations")
	fl.BoolVar(&cmd.Options.IncludeEntities, "include-entities", false, "Include GTFS entities in JSON output")
	fl.BoolVar(&cmd.Options.IncludeRouteGeometries, "include-route-geometries", false, "Include route geometries in JSON output")
	fl.BoolVar(&cmd.Options.IncludeServiceLevels, "include-service-levels", false, "Include service levels in JSON output")
	fl.BoolVar(&cmd.Options.IncludeRealtimeJson, "rt-json", false, "Include GTFS-RT proto messages as JSON in validation report")
	fl.BoolVar(&cmd.SaveValidationReport, "validation-report", false, "Save static validation report in database")
	fl.StringVar(&cmd.ValidationReportStorage, "validation-report-storage", "", "Storage path for saving validation report JSON")
	fl.IntVar(&cmd.FVID, "save-fvid", 0, "Save report to feed version ID")
	fl.StringSliceVar(&cmd.rtFiles, "rt", nil, "Include GTFS-RT proto message in validation report")
	fl.IntVar(&cmd.Options.ErrorLimit, "error-limit", 1000, "Max number of detailed errors per error group")
	fl.StringSliceVar(&cmd.errorThresholds, "error-threshold", nil, "Fail validation if file exceeds error percentage; format: 'filename:percent' or '*:percent' for default (e.g., 'stops.txt:5' or '*:10')")
	fl.StringVar(&cmd.SecretsFile, "secrets", "", "Path to DMFR Secrets file (requires --dmfr and --feed-id)")
	fl.StringArrayVar(&cmd.SecretEnv, "secret-env", nil, "Specify secret from environment variable as feed_id:ENV_VAR or file.json:ENV_VAR (requires --dmfr and --feed-id)")
	fl.StringVar(&cmd.DMFRFile, "dmfr", "", "DMFR file providing feed URL and authorization config; used with --feed-id")
	fl.StringVar(&cmd.FeedID, "feed-id", "", "Feed onestop ID for DMFR and secret lookup (requires --dmfr)")
	fl.StringVar(&cmd.URLType, "url-type", "static_current", "URL type in DMFR feed.urls to validate")
	fl.BoolVar(&cmd.AllowFTPFetch, "allow-ftp-fetch", false, "Allow fetching from FTP urls when --dmfr is used")
	fl.BoolVar(&cmd.AllowLocalFetch, "allow-local-fetch", false, "Allow fetching from filesystem paths when --dmfr is used")
	fl.BoolVar(&cmd.AllowS3Fetch, "allow-s3-fetch", false, "Allow fetching from S3 urls when --dmfr is used")
	fl.BoolVar(&cmd.AllowHTTPFetchUnfiltered, "allow-http-fetch-unfiltered", false, "Disable SSRF protection for http(s) fetches; allow private/loopback/metadata IPs (use only for local CLI runs)")
}

func (cmd *ValidatorCommand) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	if fl.NArg() >= 1 {
		cmd.readerPath = fl.Arg(0)
	}
	if cmd.DMFRFile != "" && cmd.FeedID == "" {
		return errors.New("--dmfr requires --feed-id")
	}
	if cmd.FeedID != "" && cmd.DMFRFile == "" {
		return errors.New("--feed-id requires --dmfr")
	}
	if (cmd.SecretsFile != "" || len(cmd.SecretEnv) > 0) && cmd.DMFRFile == "" {
		return errors.New("--secrets and --secret-env require --dmfr and --feed-id")
	}
	if cmd.readerPath == "" && cmd.DMFRFile == "" {
		return errors.New("requires input reader or --dmfr with --feed-id")
	}
	if cmd.SecretsFile != "" {
		r, err := dmfr.LoadAndParseRegistry(cmd.SecretsFile)
		if err != nil {
			return err
		}
		cmd.secrets = r.Secrets
	}
	for _, se := range cmd.SecretEnv {
		secret, err := parseSecretEnv(se)
		if err != nil {
			return err
		}
		cmd.secrets = append(cmd.secrets, secret)
	}
	cmd.Options.ValidateRealtimeMessages = cmd.rtFiles
	cmd.Options.ExtensionDefs = cmd.extensionDefs
	cmd.Options.EvaluateAt = time.Now().In(time.UTC)
	if len(cmd.errorThresholds) > 0 {
		thresholds, err := parseErrorThresholds(cmd.errorThresholds)
		if err != nil {
			return err
		}
		cmd.Options.ErrorThreshold = thresholds
	}

	// Output to stdout implies quiet mode
	if cmd.OutputFile == "-" {
		cmd.Quiet = true
	}

	// Suppress logs when quiet mode is enabled
	// TODO: Remove direct zerolog import once log package exports level constants
	if cmd.Quiet {
		log.SetLevel(zerolog.FatalLevel)
	}

	return nil
}

func (cmd *ValidatorCommand) Run(ctx context.Context) error {
	if cmd.DMFRFile != "" {
		tmpfile, err := cmd.fetchWithAuth(ctx)
		if err != nil {
			return err
		}
		// tmpfile may carry a "#fragment" for the reader; strip it before unlinking.
		removePath, _, _ := strings.Cut(tmpfile, "#")
		defer os.Remove(removePath)
		cmd.readerPath = tmpfile
	}
	// Only log if not outputting JSON to stdout
	if cmd.shouldShowLogs() {
		log.For(ctx).Info().Msgf("Validating: %s", cmd.readerPath)
	}
	reader, err := ext.OpenReader(cmd.readerPath)
	if err != nil {
		return err
	}
	defer reader.Close()
	v, err := validator.NewValidator(reader, cmd.Options)
	if err != nil {
		return err
	}
	result, err := v.Validate(ctx)
	if err != nil {
		return err
	}

	// Write JSON output if -o flag specified
	if cmd.OutputFile != "" {
		b, err := json.MarshalIndent(snakejson.SnakeMarshaller{Value: result}, "", "  ")
		if err != nil {
			return err
		}

		outf := os.Stdout
		if cmd.OutputFile != "-" {
			var err error
			outf, err = os.Create(cmd.OutputFile)
			if err != nil {
				return err
			}
			defer outf.Close()
		}
		if _, err := outf.Write(b); err != nil {
			return err
		}
		if _, err := outf.Write([]byte("\n")); err != nil {
			return err
		}
	}

	// Save to database
	if cmd.SaveValidationReport {
		if cmd.shouldShowLogs() {
			log.For(ctx).Info().Msgf("Saving validation report to feed version: %d", cmd.FVID)
		}
		writer, err := tldb.OpenWriter(cmd.DBURL, true)
		if err != nil {
			return err
		}
		atx := writer.Adapter
		defer atx.Close()
		if err := validator.SaveValidationReport(ctx, atx, result, cmd.FVID, cmd.ValidationReportStorage); err != nil {
			return err
		}
	}
	return nil
}

// fetchWithAuth downloads the feed with DMFR auth applied and returns a temp
// path the caller must remove.
func (cmd *ValidatorCommand) fetchWithAuth(ctx context.Context) (string, error) {
	reg, err := dmfr.LoadAndParseRegistry(cmd.DMFRFile)
	if err != nil {
		return "", err
	}
	var feed *dmfr.Feed
	for i := range reg.Feeds {
		if reg.Feeds[i].FeedID == cmd.FeedID {
			feed = &reg.Feeds[i]
			break
		}
	}
	if feed == nil {
		return "", fmt.Errorf("feed %q not found in %s", cmd.FeedID, cmd.DMFRFile)
	}
	// LoadAndParseRegistry doesn't populate feed.File; set it so filename-keyed
	// secrets (e.g. {"filename": "wmata.com.dmfr.json"}) resolve.
	feed.File = filepath.Base(cmd.DMFRFile)
	feedURL := cmd.readerPath
	if feedURL == "" {
		feedURL, err = urlForType(feed.URLs, cmd.URLType)
		if err != nil {
			return "", err
		}
	}
	if feedURL == "" {
		return "", fmt.Errorf("no %s URL found for feed %q", cmd.URLType, cmd.FeedID)
	}
	// Strip any "#subdir" fragment for the download and re-attach it to the
	// temp path so tlcsv's internal-zip-path semantics are preserved.
	fetchURL, fragment, hasFragment := strings.Cut(feedURL, "#")
	var reqOpts []request.RequestOption
	if cmd.AllowFTPFetch {
		reqOpts = append(reqOpts, request.WithAllowFTP)
	}
	if cmd.AllowLocalFetch {
		reqOpts = append(reqOpts, request.WithAllowLocal)
	}
	if cmd.AllowS3Fetch {
		reqOpts = append(reqOpts, request.WithAllowS3)
	}
	if cmd.AllowHTTPFetchUnfiltered {
		reqOpts = append(reqOpts, request.WithAllowHTTPUnfiltered)
	}
	if feed.Authorization.Type != "" {
		secret, err := feed.MatchSecrets(cmd.secrets, cmd.URLType)
		if err != nil {
			return "", fmt.Errorf("authorization %q configured for feed %q but %w", feed.Authorization.Type, cmd.FeedID, err)
		}
		reqOpts = append(reqOpts, request.WithAuth(secret, feed.Authorization))
	}
	if cmd.shouldShowLogs() {
		log.For(ctx).Info().Str("feed_id", cmd.FeedID).Str("url", feedURL).Str("auth_type", feed.Authorization.Type).Msg("Fetching feed")
	}
	tmpfile, fr, err := request.AuthenticatedRequestDownload(ctx, fetchURL, reqOpts...)
	if err != nil {
		if tmpfile != "" {
			os.Remove(tmpfile)
		}
		return "", err
	}
	if fr.FetchError != nil {
		if tmpfile != "" {
			os.Remove(tmpfile)
		}
		return "", fmt.Errorf("fetch failed: %w", fr.FetchError)
	}
	if hasFragment {
		return tmpfile + "#" + fragment, nil
	}
	return tmpfile, nil
}

func urlForType(urls dmfr.FeedUrls, urlType string) (string, error) {
	switch urlType {
	case "", "static_current":
		return urls.StaticCurrent, nil
	case "realtime_trip_updates":
		return urls.RealtimeTripUpdates, nil
	case "realtime_vehicle_positions":
		return urls.RealtimeVehiclePositions, nil
	case "realtime_alerts":
		return urls.RealtimeAlerts, nil
	case "gbfs_auto_discovery":
		return urls.GbfsAutoDiscovery, nil
	case "mds_provider":
		return urls.MdsProvider, nil
	}
	return "", fmt.Errorf("unsupported --url-type %q", urlType)
}
