package main

import (
	"errors"
	"flag"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/gotransit/dmfr"
	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/internal/log"
)

type dmfrCommand struct {
	args []string
}

func (cmd *dmfrCommand) run(args []string) {
	fl := flag.NewFlagSet("dmfr", flag.ExitOnError)
	fl.Parse(args)
	cmd.args = fl.Args()
	subc := args[0]
	var err error
	switch subc {
	case "validate":
		err = cmd.cmdValidate(args[1:])
	case "merge":
		err = cmd.cmdMerge(args[1:])
	case "sync":
		err = cmd.cmdImport(args[1], args[2:])
	case "fetchfeedversions":
		err = cmd.cmdFetchFeedVersions(args[1], args[2:])
	default:
		exit("Invalid subcommand: %q", subc)
	}
	if err != nil {
		exit(err.Error())
	}
}

func (cmd *dmfrCommand) cmdFetchFeedVersions(dburl string, feedids []string) error {
	writer := MustGetDBWriter(dburl, true)
	return writer.Adapter.Tx(func(atx gtdb.Adapter) error {
		found := []int64{}
		var err error
		var q sq.SelectBuilder
		if len(feedids) == 0 {
			q = atx.Sqrl().Select("id").From("current_feeds").Where("deleted_at IS NULL")
		} else {
			q = atx.Sqrl().Select("id").From("current_feeds").Where("deleted_at IS NULL").Where(sq.Eq{"onestop_id": feedids})
		}
		qstr, qargs := q.MustSql()
		err = atx.Select(&found, qstr, qargs...)
		if err != nil {
			return err
		}
		log.Info("Fetching %d feeds", len(found))
		for _, fid := range found {
			dmfr.MainFetchFeed(atx, int(fid))
		}
		return nil
	})
}

func (cmd *dmfrCommand) cmdImport(dburl string, filenames []string) error {
	writer := MustGetDBWriter(dburl, true)
	log.Info("Syncing %d DMFRs to %s", len(filenames), dburl)
	return writer.Adapter.Tx(func(atx gtdb.Adapter) error {
		dmfr.MainSync(atx, filenames)
		return nil
	})
}

func (cmd *dmfrCommand) cmdValidate(filenames []string) error {
	for _, arg := range filenames {
		log.Info("Loading DMFR: %s", arg)
		registry, err := dmfr.LoadAndParseRegistry(arg)
		if err != nil {
			return fmt.Errorf("Error when loading DMFR: %s", err.Error())
		}
		log.Info("Success loading DMFR with %d feeds", len(registry.Feeds))
	}
	return nil
}

func (cmd *dmfrCommand) cmdMerge(filenames []string) error {
	return errors.New("not implemented")
}
