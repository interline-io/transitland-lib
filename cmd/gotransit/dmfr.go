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

func (dmfrCommand) run(args []string) {
	fl := flag.NewFlagSet("dmfr", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Println("Usage: gotransit dmfr <command> [<args>]")
		fmt.Println("dmfr commands:")
		fmt.Println("  validate")
		fmt.Println("  merge")
		fmt.Println("  sync")
		fmt.Println("  fetchfeedversions")
	}
	fl.Parse(args)
	subc := fl.Arg(0)
	if subc == "" {
		fl.Usage()
		exit("")
	}
	var err error
	switch subc {
	case "validate":
		err = dmfrValidateCommand{}.run(args[1:])
	case "merge":
		err = dmfrMergeCommand{}.run(args[1:])
	case "sync":
		err = dmfrSyncCommand{}.run(args[1:])
	case "fetchfeedversions":
		err = dmfrFetchFeedVersionsCommand{}.run(args[1:])
	default:
		exit("Invalid command: %q", subc)
	}
	if err != nil {
		exit(err.Error())
	}
}

/////

type dmfrFetchFeedVersionsCommand struct{}

func (dmfrFetchFeedVersionsCommand) run(args []string) error {
	fl := flag.NewFlagSet("fetchfeedversions", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Println("Usage: gotransit dmfr fetchfeedversions <dburl> <outpath> [feedids...]")
	}
	fl.Parse(args)
	if fl.NArg() < 2 {
		fl.Usage()
		return nil
	}
	dburl := fl.Arg(0)
	outpath := fl.Arg(1)
	feedids := []string{}
	if fl.NArg() > 2 {
		feedids = fl.Args()[2:]
	}
	writer := MustGetDBWriter(dburl, true)
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
				fr, fe = dmfr.MainFetchFeed(atx, feed.ID, outpath)
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
	workers := 4
	jobs := make(chan dmfr.Feed, len(feeds))
	results := make(chan dmfr.FetchResult, len(feeds))
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go worker(w, dburl, jobs, results, &wg)
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
