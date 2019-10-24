package main

import (
	"errors"
	"flag"
	"fmt"
	"sync"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/gotransit/dmfr"
	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/internal/log"
)

type dmfrCommand struct{}

func (dmfrCommand) run(args []string) error {
	fl := flag.NewFlagSet("dmfr", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Println("Usage: gotransit dmfr <command> [<args>]")
		fmt.Println("dmfr commands:")
		fmt.Println("  validate")
		fmt.Println("  merge")
		fmt.Println("  sync")
		fmt.Println("  import")
		fmt.Println("  fetchfeedversions")
	}
	fl.Parse(args)
	subc := fl.Arg(0)
	if subc == "" {
		fl.Usage()
		exit("")
	}
	var err error
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
	case "fetchfeedversions":
		r = &dmfrFetchFeedVersionsCommand{}
	default:
		exit("Invalid command: %q", subc)
	}
	if err != nil {
		exit(err.Error())
	}
	if r == nil {
		exit("no runner!")
	}
	return r.run(fl.Args())
}

/////

type dmfrImportCommand struct {
	workers int
	limit   int
	dburl   string
	gtfsdir string
	feedids []string
}

func (cmd *dmfrImportCommand) run(args []string) error {
	fl := flag.NewFlagSet("import", flag.ExitOnError)
	fl.IntVar(&cmd.workers, "workers", 1, "Worker threads")
	fl.StringVar(&cmd.dburl, "dburl", "", "Database URL")
	fl.StringVar(&cmd.gtfsdir, "gtfsdir", ".", "GTFS Directory")
	fl.IntVar(&cmd.limit, "limit", 0, "Import at most n feeds")
	fl.Usage = func() {
		fmt.Println("Usage: gotransit dmfr import [feedids...]")
	}
	fl.Parse(args[1:])
	cmd.feedids = fl.Args()
	writer := MustGetDBWriter(cmd.dburl, true)
	defer writer.Close()
	// Get feeds
	fvids, err := dmfr.FindImportableFeeds(writer.Adapter)
	if err != nil {
		return err
	}
	if cmd.limit > 0 && cmd.limit < len(fvids) {
		fvids = fvids[:cmd.limit]
	}
	///////////////
	// Here we go
	log.Info("Importing %d feed versions", len(fvids))
	worker := func(id int, dburl string, jobs <-chan int, results chan<- error, wg *sync.WaitGroup) {
		w := MustGetDBWriter(dburl, true)
		defer writer.Close()
		for fvid := range jobs {
			results <- dmfr.MainImportFeedVersion(w.Adapter, fvid)
		}
		wg.Done()
	}
	var wg sync.WaitGroup
	jobs := make(chan int, len(fvids))
	results := make(chan error, len(fvids))
	for w := 0; w < cmd.workers; w++ {
		wg.Add(1)
		go worker(w, cmd.dburl, jobs, results, &wg)
	}
	for _, fvid := range fvids {
		jobs <- fvid
	}
	close(jobs)
	wg.Wait()
	close(results)
	// Read out results
	for err := range results {
		fmt.Println(err)
	}
	return nil
}

/////

type dmfrFetchFeedVersionsCommand struct {
	workers int
	limit   int
	dburl   string
	gtfsdir string
	feedids []string
}

func (cmd *dmfrFetchFeedVersionsCommand) run(args []string) error {
	fl := flag.NewFlagSet("fetchfeedversions", flag.ExitOnError)
	fl.IntVar(&cmd.workers, "workers", 1, "Worker threads")
	fl.IntVar(&cmd.limit, "limit", 0, "Fetch at most n feeds")
	fl.StringVar(&cmd.dburl, "dburl", "", "Database URL")
	fl.StringVar(&cmd.gtfsdir, "gtfsdir", ".", "GTFS Directory")
	fl.Usage = func() {
		fmt.Println("Usage: gotransit dmfr fetchfeedversions [feedids...]")
	}
	fl.Parse(args)
	feedids := fl.Args()
	writer := MustGetDBWriter(cmd.dburl, true)
	defer writer.Close()
	// Get feeds
	q := writer.Adapter.Sqrl().
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
	feeds := []dmfr.Feed{}
	err = writer.Adapter.Select(&feeds, qstr, qargs...)
	if err != nil {
		return err
	}
	if cmd.limit > 0 && cmd.limit < len(feeds) {
		feeds = feeds[:cmd.limit]
	}
	///////////////
	// Here we go
	log.Info("Fetching %d feeds", len(feeds))
	worker := func(id int, dburl string, jobs <-chan dmfr.Feed, results chan<- dmfr.FetchResult, wg *sync.WaitGroup) {
		w := MustGetDBWriter(dburl, true)
		defer writer.Close()
		for feed := range jobs {
			var fr dmfr.FetchResult
			err := w.Adapter.Tx(func(atx gtdb.Adapter) error {
				var fe error
				fr, fe = dmfr.MainFetchFeed(atx, feed.ID, cmd.gtfsdir)
				return fe
			})
			if err != nil {
				panic(err)
			}
			results <- fr
		}
		wg.Done()
	}

	var wg sync.WaitGroup
	jobs := make(chan dmfr.Feed, len(feeds))
	results := make(chan dmfr.FetchResult, len(feeds))
	for w := 0; w < cmd.workers; w++ {
		wg.Add(1)
		go worker(w, cmd.dburl, jobs, results, &wg)
	}
	for _, feed := range feeds {
		jobs <- feed
	}
	close(jobs)
	wg.Wait()
	close(results)

	// Read out results
	fetchNew := 0
	fetchFound := 0
	fetchErrs := 0
	for fr := range results {
		if fr.FetchError != nil {
			log.Info("Feed %s (%d): url: %s error: %s", "", fr.FeedVersion.FeedID, fr.FeedVersion.URL, fr.FetchError.Error())
			fetchErrs++
		} else if fr.Found {
			log.Info("Feed %s (%d): url: %s found: %s (%d)", "", fr.FeedVersion.FeedID, fr.FeedVersion.URL, fr.FeedVersion.SHA1, fr.FeedVersion.ID)
			fetchFound++
		} else {
			log.Info("Feed %s (%d): url: %s new: %s (%d)", "", fr.FeedVersion.FeedID, fr.FeedVersion.URL, fr.FeedVersion.SHA1, fr.FeedVersion.ID)
			fetchNew++
		}
	}
	log.Info("Existing: %d New: %d Errors: %d", fetchFound, fetchNew, fetchErrs)
	return err
}

/////

type dmfrSyncCommand struct{}

func (dmfrSyncCommand) run(args []string) error {
	fl := flag.NewFlagSet("sync", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Println("Usage: gotransit dmfr sync <dburl> <filenames...>")
	}
	fl.Parse(args)
	if fl.NArg() < 2 {
		fl.Usage()
		return nil
	}
	dburl := fl.Arg(0)
	filenames := fl.Args()[1:]
	writer := MustGetDBWriter(dburl, true)
	return writer.Adapter.Tx(func(atx gtdb.Adapter) error {
		_, err := dmfr.MainSync(atx, filenames)
		return err
	})
}

/////

type dmfrValidateCommand struct{}

func (dmfrValidateCommand) run(args []string) error {
	fl := flag.NewFlagSet("validate", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Println("Usage: gotransit dmfr validate <filenames...>")
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
		registry, err := dmfr.LoadAndParseRegistry(filename)
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

func (dmfrMergeCommand) run(args []string) error {
	return errors.New("not implemented")
}
