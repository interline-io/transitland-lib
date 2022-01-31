package sync

import (
	"fmt"
	"path/filepath"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tldb"
)

// Options sets options for a sync operation.
type Options struct {
	Filenames           []string
	HideUnseen          bool
	HideUnseenOperators bool
}

// Result is the result of a sync operation.
type Result struct {
	FeedIDs         []int
	OperatorIDs     []int
	Errors          []error
	HiddenCount     int
	HiddenOperators int
}

// MainSync .
func MainSync(atx tldb.Adapter, opts Options) (Result, error) {
	sr := Result{}
	// Load Feeds
	for _, fn := range opts.Filenames {
		reg, err := dmfr.LoadAndParseRegistry(fn)
		if err != nil {
			log.Error("%s: Error parsing DMFR: %s", fn, err.Error())
			sr.Errors = append(sr.Errors, err)
			continue
		}
		for _, rfeed := range reg.Feeds {
			fsid := rfeed.FeedID
			rfeed.File = filepath.Base(fn)
			rfeed.DeletedAt = tl.OTime{Valid: false}
			feedid, found, updated, err := UpdateFeed(atx, rfeed)
			if err != nil {
				log.Error("%s: error on feed %d: %s", fn, feedid, err)
				sr.Errors = append(sr.Errors, err)
				continue
			}
			if found && updated {
				log.Info("%s: updated feed %s (id:%d)", fn, fsid, feedid)
			} else if found {
				log.Info("%s: no changes for feed %s (id:%d)", fn, fsid, feedid)
			} else {
				log.Info("%s: new feed %s (id:%d)", fn, fsid, feedid)
			}
			sr.FeedIDs = append(sr.FeedIDs, feedid)
		}
	}
	// Load Operators
	for _, fn := range opts.Filenames {
		reg, err := dmfr.LoadAndParseRegistry(fn)
		if err != nil {
			log.Error("%s: Error parsing DMFR: %s", fn, err.Error())
			sr.Errors = append(sr.Errors, err)
			continue
		}
		for _, operator := range reg.Operators {
			osid := operator.OnestopID.String
			operator.File = tl.NewOString(filepath.Base(fn))
			operator.DeletedAt = tl.OTime{Valid: false}
			operatorid, found, updated, err := UpdateOperator(atx, operator)
			if err != nil {
				log.Error("%s: error on operator %s: %s", fn, osid, err)
				sr.Errors = append(sr.Errors, err)
				continue
			}
			if found && updated {
				log.Info("%s: updated operator %s (id:%d)", fn, osid, operatorid)
			} else if found {
				log.Info("%s: no changes for operator %s (id:%d)", fn, osid, operatorid)
			} else {
				log.Info("%s: new operator %s (id:%d)", fn, osid, operatorid)
			}
			sr.OperatorIDs = append(sr.OperatorIDs, operatorid)
		}
	}
	// Rollback on any errors
	if len(sr.Errors) > 0 {
		log.Error("Rollback due to one or more failures")
		return sr, fmt.Errorf("failed: %s", sr.Errors[0].Error())
	}
	// Hide
	if opts.HideUnseen {
		var err error
		sr.HiddenCount, err = HideUnseedFeeds(atx, sr.FeedIDs)
		if err != nil {
			log.Error("Error soft-deleting feeds: %s", err.Error())
			return sr, err
		}
		if sr.HiddenCount > 0 {
			log.Info("Soft-deleted %d feeds", sr.HiddenCount)
		}
	}
	if opts.HideUnseenOperators {
		var err error
		sr.HiddenOperators, err = HideUnseedOperators(atx, sr.OperatorIDs)
		if err != nil {
			log.Error("Error soft-deleting operators: %s", err.Error())
			return sr, err
		}
		if sr.HiddenOperators > 0 {
			log.Info("Soft-deleted %d operators", sr.HiddenOperators)
		}
	}
	// Update any automatically generated agency-operator associations
	if err := UpdateFeedGeneratedOperators(atx, sr.FeedIDs); err != nil {
		sr.Errors = append(sr.Errors, err)
	}
	// Rollback on any errors
	if len(sr.Errors) > 0 {
		log.Error("Rollback due to one or more failures")
		return sr, fmt.Errorf("failed: %s", sr.Errors[0].Error())
	}
	return sr, nil
}
