package dmfr

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/internal/log"
)

// Command is the main entry point to the DMFR command
type Command struct {
	test int
}

// Run the DMFR command.
func (cmd *Command) Run(args []string) error {
	fl := flag.NewFlagSet("dmfr", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Println("Usage: dmfr <command> [<args>]")
		fmt.Println("dmfr commands:")
		fmt.Println("  validate")
		fmt.Println("  merge")
		fmt.Println("  sync")
		fmt.Println("  import")
		fmt.Println("  fetch")
		fl.PrintDefaults()
	}
	fl.Parse(args)
	subc := fl.Arg(0)
	if subc == "" {
		fl.Usage()
		return nil
	}
	type runner interface {
		Run([]string) error
	}
	var r runner
	switch subc {
	case "validate":
		r = &dmfrValidateCommand{}
	case "merge":
		r = &dmfrMergeCommand{}
	case "sync":
		r = &dmfrSyncCommand{}
	case "import":
		r = &dmfrImportCommand{}
	case "fetch":
		r = &dmfrFetchCommand{}
	default:
		return fmt.Errorf("Invalid command: %q", subc)
	}
	return r.Run(fl.Args()[1:]) // consume first arg
}

/////

type dmfrImportCommand struct {
	workers    int
	limit      uint64
	dburl      string
	gtfsdir    string
	coverdate  string
	dryrun     bool
	feedids    []string
	extensions arrayFlags
	adapter    gtdb.Adapter // allow for mocks
}

func (cmd *dmfrImportCommand) Run(args []string) error {
	fl := flag.NewFlagSet("import", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Println("Usage: import [feedids...]")
		fl.PrintDefaults()
	}
	fl.Var(&cmd.extensions, "ext", "Include GTFS Extension")
	fl.IntVar(&cmd.workers, "workers", 1, "Worker threads")
	fl.StringVar(&cmd.dburl, "dburl", os.Getenv("DMFR_DATABASE_URL"), "Database URL ($DMFR_DATABASE_URL)")
	fl.StringVar(&cmd.gtfsdir, "gtfsdir", ".", "GTFS Directory")
	fl.StringVar(&cmd.coverdate, "date", "", "Service on date")
	fl.Uint64Var(&cmd.limit, "limit", 0, "Import at most n feeds")
	fl.BoolVar(&cmd.dryrun, "dryrun", false, "Dry run; print feeds that would be imported and exit")
	fl.Parse(args)
	cmd.feedids = fl.Args()
	if cmd.adapter == nil {
		writer := mustGetWriter(cmd.dburl, true)
		cmd.adapter = writer.Adapter
		defer writer.Close()
	}
	// Query to get FVs to import
	q := cmd.adapter.Sqrl().
		Select("feed_versions.id as feed_version_id", "feed_versions.sha1", "current_feeds.id as feed_id", "current_feeds.onestop_id").
		From("feed_versions").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id").
		LeftJoin("feed_version_gtfs_imports ON feed_versions.id = feed_version_gtfs_imports.feed_version_id").
		Where("feed_version_gtfs_imports.id IS NULL").
		OrderBy("feed_versions.id")
	if cmd.limit > 0 {
		// Max feeds
		q = q.Limit(cmd.limit)
	}
	if len(cmd.feedids) > 0 {
		// Limit to specified feeds
		q = q.Where(sq.Eq{"onestop_id": cmd.feedids})
	}
	if cmd.coverdate == "" {
		// Set default date
		cmd.coverdate = time.Now().Format("2006-01-02")
	}
	if cmd.coverdate != "" {
		// Limit to service date
		q = q.
			Where(sq.LtOrEq{"feed_versions.earliest_calendar_date": cmd.coverdate}).
			Where(sq.GtOrEq{"feed_versions.latest_calendar_date": cmd.coverdate})
	}
	qstr, qargs, err := q.ToSql()
	if err != nil {
		return err
	}
	type qr struct { // hold results
		FeedVersionID int
		FeedID        int
		SHA1          string
		OnestopID     string
	}
	qrs := []qr{}
	err = cmd.adapter.Select(&qrs, qstr, qargs...)
	if err != nil {
		return err
	}
	qlookup := map[int]qr{}
	for _, i := range qrs {
		qlookup[i.FeedVersionID] = i
	}
	///////////////
	// Here we go
	log.Info("Importing %d feed versions", len(qlookup))
	if cmd.dryrun {
		for fvid, i := range qlookup {
			log.Info("Feed %s (id:%d): FeedVersion %s (id:%d): dry-run", i.OnestopID, i.FeedID, i.SHA1, fvid)
		}
		return nil
	}
	var wg sync.WaitGroup
	jobs := make(chan ImportOptions, len(qlookup))
	results := make(chan ImportResult, len(qlookup))
	for w := 0; w < cmd.workers; w++ {
		wg.Add(1)
		go dmfrImportWorker(w, cmd.adapter, jobs, results, &wg)
	}
	for fvid := range qlookup {
		jobs <- ImportOptions{FeedVersionID: fvid, Directory: cmd.gtfsdir, Extensions: cmd.extensions}
	}
	close(jobs)
	wg.Wait()
	close(results)
	// Read out results
	for result := range results {
		fvid := result.FeedVersionImport.FeedVersionID
		i := qlookup[fvid]
		if result.FeedVersionImport.Success {
			log.Info("Feed %s (id:%d): FeedVersion %s (id:%d): success: count: %v errors: %v", i.OnestopID, i.FeedID, i.SHA1, fvid, result.FeedVersionImport.EntityCount, result.FeedVersionImport.ErrorCount)
		} else {
			log.Info("Feed %s (id:%d): FeedVersion %s (id:%d): error: %s", i.OnestopID, i.FeedID, i.SHA1, i.SHA1, result.FeedVersionImport.FeedVersionID, err.Error())
		}
	}
	return nil
}

func dmfrImportWorker(id int, adapter gtdb.Adapter, jobs <-chan ImportOptions, results chan<- ImportResult, wg *sync.WaitGroup) {
	for opts := range jobs {
		fviresult, err := MainImportFeedVersion(adapter, opts)
		if err != nil {
			log.Info("Error: %s", err.Error())
		}
		results <- fviresult
	}
	wg.Done()
}

/////

type dmfrFetchCommand struct {
	workers   int
	limit     int
	dburl     string
	gtfsdir   string
	allowdups bool
	feedids   []string
	adapter   gtdb.Adapter
}

func (cmd *dmfrFetchCommand) Run(args []string) error {
	fl := flag.NewFlagSet("fetch", flag.ExitOnError)
	fl.IntVar(&cmd.workers, "workers", 1, "Worker threads")
	fl.IntVar(&cmd.limit, "limit", 0, "Maximum number of feeds to fetch")
	fl.StringVar(&cmd.dburl, "dburl", os.Getenv("DMFR_DATABASE_URL"), "Database URL ($DMFR_DATABASE_URL)")
	fl.StringVar(&cmd.gtfsdir, "gtfsdir", ".", "GTFS Directory")
	fl.BoolVar(&cmd.allowdups, "allow-duplicate-contents", false, "Allow duplicate internal SHA1 contents")
	fl.Usage = func() {
		fmt.Println("Usage: fetch [feedids...]")
		fl.PrintDefaults()
	}
	fl.Parse(args)
	feedids := fl.Args()
	if cmd.adapter == nil {
		writer := mustGetWriter(cmd.dburl, true)
		cmd.adapter = writer.Adapter
		defer writer.Close()
	}
	// Get feeds
	q := cmd.adapter.Sqrl().
		Select("*").
		From("current_feeds").
		Where("deleted_at IS NULL").
		Where("spec = ?", "gtfs")
	if len(feedids) > 0 {
		q = q.Where(sq.Eq{"onestop_id": feedids})
	}
	qstr, qargs, err := q.ToSql()
	if err != nil {
		return err
	}
	feeds := []Feed{}
	err = cmd.adapter.Select(&feeds, qstr, qargs...)
	if err != nil {
		return err
	}
	if cmd.limit > 0 && cmd.limit < len(feeds) {
		feeds = feeds[:cmd.limit]
	}
	osids := map[int]string{}
	for _, feed := range feeds {
		osids[feed.ID] = feed.FeedID
	}
	///////////////
	// Here we go
	log.Info("Fetching %d feeds", len(feeds))
	fetchNew := 0
	fetchFound := 0
	fetchErrs := 0
	var wg sync.WaitGroup
	jobs := make(chan FetchOptions, len(feeds))
	results := make(chan FetchResult, len(feeds))
	for w := 0; w < cmd.workers; w++ {
		wg.Add(1)
		go dmfrFetchWorker(w, cmd.adapter, jobs, results, &wg)
	}
	for _, feed := range feeds {
		opts := FetchOptions{
			FeedID:                  feed.ID,
			Directory:               cmd.gtfsdir,
			IgnoreDuplicateContents: cmd.allowdups,
		}
		jobs <- opts
	}
	close(jobs)
	wg.Wait()
	close(results)
	for fr := range results {
		osid := osids[fr.FeedVersion.FeedID]
		if err != nil {
			log.Info("Feed %s (id:%d): url: %s critical error: %s", osid, fr.FeedVersion.FeedID, fr.FeedVersion.URL, err.Error())
			fetchErrs++
		} else if fr.FetchError != nil {
			log.Info("Feed %s (id:%d): url: %s fetch error: %s", osid, fr.FeedVersion.FeedID, fr.FeedVersion.URL, fr.FetchError.Error())
			fetchErrs++
		} else if fr.FoundSHA1 {
			log.Info("Feed %s (id:%d): url: %s found zip sha1: %s (id:%d)", osid, fr.FeedVersion.FeedID, fr.FeedVersion.URL, fr.FeedVersion.SHA1, fr.FeedVersion.ID)
			fetchFound++
		} else if fr.FoundDirSHA1 {
			log.Info("Feed %s (id:%d): url: %s found contents sha1: %s (id:%d)", osid, fr.FeedVersion.FeedID, fr.FeedVersion.URL, fr.FeedVersion.SHA1Dir, fr.FeedVersion.ID)
			fetchFound++
		} else {
			log.Info("Feed %s (id:%d): url: %s new: %s (id:%d)", osid, fr.FeedVersion.FeedID, fr.FeedVersion.URL, fr.FeedVersion.SHA1, fr.FeedVersion.ID)
			fetchNew++
		}
	}
	log.Info("Existing: %d New: %d Errors: %d", fetchFound, fetchNew, fetchErrs)
	return nil
}

func dmfrFetchWorker(id int, adapter gtdb.Adapter, jobs <-chan FetchOptions, results chan<- FetchResult, wg *sync.WaitGroup) {
	for opts := range jobs {
		var fr FetchResult
		err := adapter.Tx(func(atx gtdb.Adapter) error {
			var fe error
			fr, fe = MainFetchFeed(atx, opts)
			return fe
		})
		if err != nil {
			fmt.Println("Critical error:", err)
		}
		results <- fr
	}
	wg.Done()
}

/////

type dmfrSyncCommand struct {
	dburl     string
	filenames []string
	adapter   gtdb.Adapter // allow for mocks
}

func (cmd *dmfrSyncCommand) Run(args []string) error {
	fl := flag.NewFlagSet("sync", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Println("Usage: sync <filenames...>")
		fl.PrintDefaults()
	}
	fl.StringVar(&cmd.dburl, "dburl", os.Getenv("DMFR_DATABASE_URL"), "Database URL (default: $DMFR_DATABASE_URL)")
	fl.Parse(args)
	cmd.filenames = fl.Args()
	if cmd.adapter == nil {
		writer := mustGetWriter(cmd.dburl, true)
		cmd.adapter = writer.Adapter
		defer writer.Close()
	}
	return cmd.adapter.Tx(func(atx gtdb.Adapter) error {
		_, err := MainSync(atx, cmd.filenames)
		return err
	})
}

/////

type dmfrValidateCommand struct{}

func (dmfrValidateCommand) Run(args []string) error {
	fl := flag.NewFlagSet("validate", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Println("Usage: validate <filenames...>")
		fl.PrintDefaults()
	}
	fl.Parse(args)
	if fl.NArg() == 0 {
		fl.Usage()
		return nil
	}
	filenames := fl.Args()
	errs := []error{}
	for _, filename := range filenames {
		log.Info("Loading DMFR: %s", filename)
		registry, err := LoadAndParseRegistry(filename)
		if err != nil {
			errs = append(errs, err)
			log.Info("%s: Error when loading DMFR: %s", filename, err.Error())
		} else {
			log.Info("%s: Success loading DMFR with %d feeds", filename, len(registry.Feeds))
		}
	}
	if len(errs) > 0 {
		return errors.New("")
	}
	return nil
}

/////

type dmfrMergeCommand struct{}

func (dmfrMergeCommand) Run(args []string) error {
	return errors.New("not implemented")
}

//// Util

// https://stackoverflow.com/questions/28322997/how-to-get-a-list-of-values-into-a-flag-in-golang/28323276#28323276
type arrayFlags []string

func (i *arrayFlags) String() string {
	return strings.Join(*i, ",")
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// mustGetWriter opens & creates a db writer, panic on failure
func mustGetWriter(dburl string, create bool) *gtdb.Writer {
	// Writer
	writer, err := gtdb.NewWriter(dburl)
	if err != nil {
		panic(err)
	}
	if err := writer.Open(); err != nil {
		panic(err)
	}
	if create {
		if err := writer.Create(); err != nil {
			panic(err)
		}
	}
	return writer
}
