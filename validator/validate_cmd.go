package validator

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr/store"
	"github.com/interline-io/transitland-lib/internal/cli"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/jmoiron/sqlx"
)

// Command
type Command struct {
	Options     Options
	OutputFile  string
	DBURL       string
	checkRtUrls cli.ArrayFlags
	extensions  cli.ArrayFlags

	//////
	UseHeaderTimestamp      bool
	CheckRtUrls             []string
	FeedId                  string
	Timezone                string
	ValidationReportStorage string
	ValidateRtPath          string
	ValidateStaticPath      string
	Storage                 string
	RefreshRate             int
	ForceFvid               int

	//////
	vt         *Validator
	db         sqlx.Ext
	checkFvid  sync.Mutex
	activeFvid int
}

func (cmd *Command) Parse(args []string) error {
	fl := flag.NewFlagSet("validate", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: validate [reader]")
		fl.PrintDefaults()
	}
	fl.Var(&cmd.extensions, "ext", "Include GTFS Extension")
	fl.StringVar(&cmd.OutputFile, "o", "", "Write validation report as JSON to file")
	fl.IntVar(&cmd.RefreshRate, "refresh", 0, "GTFS-RT URL Refresh rate in seconds")
	fl.IntVar(&cmd.ForceFvid, "force-fvid", 0, "force feed version id")
	fl.IntVar(&cmd.Options.ErrorLimit, "error-limit", 1000, "Max number of detailed errors per error group")
	fl.BoolVar(&cmd.Options.BestPractices, "best-practices", false, "Include Best Practices validations")
	fl.BoolVar(&cmd.Options.IncludeRealtimeJson, "rt-json", false, "Include GTFS-RT proto messages as JSON in validation report")
	fl.BoolVar(&cmd.UseHeaderTimestamp, "use-header-timestamp", false, "Use header time")
	fl.StringVar(&cmd.Storage, "storage", "", "Static storage")
	fl.StringVar(&cmd.ValidationReportStorage, "validation-report-storage", "", "Storage path for saving validation report JSON")
	fl.StringVar(&cmd.ValidateRtPath, "validate-rt-path", "", "Validate messages in directory")
	fl.StringVar(&cmd.Timezone, "timezone", "America/Los_Angeles", "Timezone")
	fl.StringVar(&cmd.FeedId, "feed", "", "Feed")
	fl.Var(&cmd.checkRtUrls, "rt", "Include GTFS-RT proto message file or URL in validation report")
	err := fl.Parse(args)
	if err != nil {
		fl.Usage()
		return err
	}
	if fl.NArg() > 1 {
		return errors.New("unknown argument")
	} else if fl.NArg() == 1 {
		cmd.ValidateStaticPath = fl.Arg(0)
	}
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	cmd.CheckRtUrls = append(cmd.CheckRtUrls, cmd.checkRtUrls...)
	cmd.Options.Extensions = append(cmd.Options.Extensions, cmd.extensions...)
	return nil
}

func (cmd *Command) Run() error {
	// Open database if provided
	if cmd.DBURL != "" {
		r, err := tldb.NewReader(cmd.DBURL)
		if err != nil {
			return err
		}
		if err := r.Open(); err != nil {
			return err
		}
		cmd.db = r.Adapter.DBX()
	}

	// Check local RTs
	var initialRtChecks []string
	initialRtChecks = append(initialRtChecks, cmd.CheckRtUrls...)
	if cmd.ValidateRtPath != "" {
		localRtFiles, err := getFiles(cmd.ValidateRtPath)
		if err != nil {
			return err
		}
		initialRtChecks = append(initialRtChecks, localRtFiles...)
	}

	// Initial check feed
	if vtReport, err := cmd.checkFeed(initialRtChecks); err != nil {
		_ = vtReport
		return err
	} else {
		if cmd.OutputFile != "" {
			// Save to output file
		}
	}

	// for _, localRtFile := range localRtFiles {
	// 	now, nowLocal := cmd.now()
	// 	if cmd.UseHeaderTimestamp {
	// 		rtMsg, err := rt.ReadFile(localRtFile)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		now := time.Unix(int64(rtMsg.GetHeader().GetTimestamp()), 0)
	// 		nowLocal = now.In(nowLocal.Location())
	// 	}
	// 	if err := cmd.pollRt([]string{localRtFile}, now, nowLocal); err != nil {
	// 		log.Error().Err(err).Msg("Failed to check rt")
	// 		return err
	// 	}
	// }

	// Check RT urls
	// if len(cmd.CheckRtUrls) > 0 {
	// 	now, nowLocal := cmd.now()
	// 	if err := cmd.pollRt(cmd.CheckRtUrls, now, nowLocal); err != nil {
	// 		log.Error().Err(err).Msg("Failed to check rt")
	// 		return err
	// 	}
	// }

	// Poll RT urls
	if cmd.RefreshRate > 0 {
		exit := make(chan string)
		ticker := time.NewTicker(time.Duration(cmd.RefreshRate) * time.Second)
		go func() {
			for ; true; <-ticker.C {
				if _, err := cmd.checkFeed(nil); err != nil {
					log.Error().Err(err).Msg("Failed to check feed")
					continue
				}
				now, nowLocal := cmd.now()
				result, err := cmd.vt.ValidateRTs(cmd.CheckRtUrls, now, nowLocal)
				if err != nil {
					log.Error().Err(err).Msg("Failed to validate RT messages")
					continue
				}

				// Save report
				if cmd.db == nil || cmd.activeFvid < 1 {
					log.Info().Msg("no database")
					continue
				}
				atx := tldb.NewPostgresAdapterFromDBX(cmd.db)
				if err := SaveValidationReport(
					atx,
					result,
					cmd.activeFvid,
					cmd.ValidationReportStorage,
				); err != nil {
					log.Error().Err(err).Msg("Failed to save validation report")
				}
				log.Info().Msg("Saved report")
			}
		}()
		<-exit
	}
	return nil

}

func (cmd *Command) checkFeed(rtUrls []string) (*Result, error) {
	log.Info().Msg("Checking static feed")
	cmd.checkFvid.Lock()
	defer cmd.checkFvid.Unlock()

	// Open static GTFS
	var reader *tlcsv.Reader
	if cmd.ValidateStaticPath != "" {
		if cmd.activeFvid == cmd.ForceFvid {
			return nil, nil
		}
		var err error
		reader, err = tlcsv.NewReader(cmd.ValidateStaticPath)
		if err != nil {
			return nil, err
		}
		cmd.activeFvid = cmd.ForceFvid
	} else if cmd.db != nil && cmd.FeedId != "" {
		// Check fv
		atx := tldb.NewPostgresAdapterFromDBX(cmd.db)
		checkFv := struct {
			ID   int
			SHA1 string
		}{}
		q := atx.Sqrl().Select("feed_versions.id, feed_versions.sha1").From("feed_versions")
		if cmd.ForceFvid > 0 {
			q = q.Where(sq.Eq{"id": cmd.ForceFvid})
		} else {
			q = q.Join("feed_states on feed_states.feed_version_id = feed_versions.id").
				Join("current_feeds on current_feeds.id = feed_versions.feed_id").
				Where("current_feeds.onestop_id = ?", cmd.FeedId)
		}
		qstr, qargs, _ := q.ToSql()
		if err := atx.Get(&checkFv, qstr, qargs...); err != nil {
			return nil, err
		}
		if cmd.activeFvid == checkFv.ID {
			return nil, nil
		}
		log.Info().Int("fvid", checkFv.ID).Str("sha1", checkFv.SHA1).Msg("got feed version")
		cmd.activeFvid = checkFv.ID
		// Fetch from storage
		fvFile := fmt.Sprintf("%s.zip", checkFv.SHA1)
		tladapter, err := store.NewStoreAdapter(cmd.Storage, fvFile, "")
		if err != nil {
			return nil, err
		}
		reader, err = tlcsv.NewReaderFromAdapter(tladapter)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("no static feed")
	}

	// Open
	if err := reader.Open(); err != nil {
		return nil, err
	}

	// Validate
	newOpts := cmd.Options
	newOpts.ValidateRealtimeMessages = rtUrls
	newVt, err := NewValidator(reader, cmd.Options)
	if err != nil {
		return nil, err
	}
	vtReport, err := newVt.Validate()
	if err != nil {
		return nil, err
	}
	cmd.vt = newVt
	return vtReport, nil
}

func (cmd *Command) now() (time.Time, time.Time) {
	// Get local time
	tz, err := time.LoadLocation(cmd.Timezone)
	if err != nil {
		panic(err)
	}
	now := time.Now()
	nowLocal := now.In(tz)
	return now, nowLocal
}

func getFiles(path string) ([]string, error) {
	files := []string{}
	if err := filepath.Walk(path,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if strings.HasSuffix(info.Name(), ".pb") {
				files = append(files, path)
			}
			return nil
		}); err != nil {
		return nil, err
	}
	return files, nil
}
