package main

import (
	"errors"
	"flag"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/gotransit/dmfr"
	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/internal/log"
)

type dmfrCommand struct{}

func (dmfrCommand) run(args []string) {
	fl := flag.NewFlagSet("dmfr", flag.ExitOnError)
	fl.Parse(args)
	if len(args) == 0 {
		exit("subcommand needed")
	}
	subc := args[0]
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
		exit("Invalid subcommand: %q", subc)
	}
	if err != nil {
		exit(err.Error())
	}
}

/////

type dmfrFetchFeedVersionsCommand struct{}

func (dmfrFetchFeedVersionsCommand) run(args []string) error {
	if len(args) < 2 {
		exit("<dburl> <outpath> [feedids...]")
	}
	dburl := args[0]
	outpath := args[1]
	feedids := args[2:]
	writer := MustGetDBWriter(dburl, true)
	//
	fetchNew := []string{}
	fetchFound := []string{}
	fetchErrs := []error{}
	// Run inside txn, abort on serious errors
	err := writer.Adapter.Tx(func(atx gtdb.Adapter) error {
		var q sq.SelectBuilder
		if len(feedids) == 0 {
			q = atx.Sqrl().Select("id").From("current_feeds").Where("deleted_at IS NULL")
		} else {
			q = atx.Sqrl().Select("id").From("current_feeds").Where("deleted_at IS NULL").Where(sq.Eq{"onestop_id": feedids})
		}
		qstr, qargs, err := q.ToSql()
		if err != nil {
			return err
		}
		found := []int64{}
		err = atx.Select(&found, qstr, qargs...)
		if err != nil {
			return err
		}
		log.Info("Fetching %d feeds", len(found))
		for _, fid := range found {
			fv, found, err := dmfr.MainFetchFeed(atx, int(fid), outpath)
			if err != nil {
				fetchErrs = append(fetchErrs, err)
			} else if found {
				fetchFound = append(fetchFound, fv.SHA1)
			} else {
				fetchNew = append(fetchNew, fv.SHA1)
			}
		}
		return nil
	})
	log.Info("Existing: %d New: %d Errors: %d", len(fetchFound), len(fetchNew), len(fetchErrs))
	return err
}

/////

type dmfrSyncCommand struct{}

func (dmfrSyncCommand) run(args []string) error {
	if len(args) < 2 {
		exit("<dburl> <filenames...>")
	}
	dburl := args[0]
	filenames := args[1:]
	writer := MustGetDBWriter(dburl, true)
	return writer.Adapter.Tx(func(atx gtdb.Adapter) error {
		_, err := dmfr.MainSync(atx, filenames)
		return err
	})
}

/////

type dmfrValidateCommand struct{}

func (dmfrValidateCommand) run(args []string) error {
	filenames := args
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
