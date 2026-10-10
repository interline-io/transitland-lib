package sync

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
)

// Options sets options for a sync operation.
type Options struct {
	Filenames           []string
	Registries          []*dmfr.Registry // Pre-parsed registries (e.g., from stdin)
	HideUnseen          bool
	HideUnseenOperators bool
	SetPublic           *bool // nil=default (new feeds public, existing unchanged), true=force public, false=force private
}

// Result is the result of a sync operation.
type Result struct {
	FeedIDs         []int
	OperatorIDs     []int
	Errors          []error
	HiddenCount     int
	HiddenOperators int
}

func MainSync(ctx context.Context, atx tldb.Adapter, opts Options) (Result, error) {
	return Sync(ctx, atx, opts)
}

func Sync(ctx context.Context, atx tldb.Adapter, opts Options) (Result, error) {
	sr := Result{}

	// Collect all registries to process
	type regSource struct {
		name string
		reg  *dmfr.Registry
	}
	var regs []regSource

	// Load from filenames
	for _, fn := range opts.Filenames {
		reg, err := dmfr.LoadAndParseRegistry(fn)
		if err != nil {
			log.For(ctx).Error().Msgf("%s: Error parsing DMFR: %s", fn, err.Error())
			sr.Errors = append(sr.Errors, err)
			continue
		}
		regs = append(regs, regSource{name: filepath.Base(fn), reg: reg})
	}

	// Add pre-parsed registries
	for _, reg := range opts.Registries {
		regs = append(regs, regSource{name: "stdin", reg: reg})
	}

	// Load Feeds
	for _, rs := range regs {
		for _, rfeed := range rs.reg.Feeds {
			fsid := rfeed.FeedID
			rfeed.File = rs.name
			rfeed.DeletedAt = tt.Time{}
			feedid, found, updated, err := UpdateFeed(ctx, atx, rfeed, opts.SetPublic)
			if err != nil {
				log.For(ctx).Error().Msgf("%s: error on feed %d: %s", rs.name, feedid, err)
				sr.Errors = append(sr.Errors, err)
				continue
			}
			if found && updated {
				log.For(ctx).Info().Msgf("%s: updated feed %s (id:%d)", rs.name, fsid, feedid)
			} else if found {
				log.For(ctx).Info().Msgf("%s: no changes for feed %s (id:%d)", rs.name, fsid, feedid)
			} else {
				log.For(ctx).Info().Msgf("%s: new feed %s (id:%d)", rs.name, fsid, feedid)
			}
			sr.FeedIDs = append(sr.FeedIDs, feedid)
		}
	}
	// Load Operators
	for _, rs := range regs {
		for _, operator := range rs.reg.Operators {
			osid := operator.OnestopID.Val
			operator.File.Set(rs.name)
			operator.DeletedAt.Unset()
			operatorid, found, updated, err := UpdateOperator(ctx, atx, operator)
			if err != nil {
				log.For(ctx).Error().Msgf("%s: error on operator %s: %s", rs.name, osid, err)
				sr.Errors = append(sr.Errors, err)
				continue
			}
			if found && updated {
				log.For(ctx).Info().Msgf("%s: updated operator %s (id:%d)", rs.name, osid, operatorid)
			} else if found {
				log.For(ctx).Info().Msgf("%s: no changes for operator %s (id:%d)", rs.name, osid, operatorid)
			} else {
				log.For(ctx).Info().Msgf("%s: new operator %s (id:%d)", rs.name, osid, operatorid)
			}
			sr.OperatorIDs = append(sr.OperatorIDs, operatorid)
		}
	}
	// Rollback on any errors
	if len(sr.Errors) > 0 {
		log.For(ctx).Error().Msgf("Rollback due to one or more failures")
		return sr, fmt.Errorf("failed: %s", sr.Errors[0].Error())
	}
	// Hide
	if opts.HideUnseen {
		var err error
		sr.HiddenCount, err = HideUnseedFeeds(ctx, atx, sr.FeedIDs)
		if err != nil {
			log.For(ctx).Error().Msgf("Error soft-deleting feeds: %s", err.Error())
			return sr, err
		}
		if sr.HiddenCount > 0 {
			log.For(ctx).Info().Msgf("Soft-deleted %d feeds", sr.HiddenCount)
		}
	}
	if opts.HideUnseenOperators {
		var err error
		sr.HiddenOperators, err = HideUnseedOperators(atx, sr.OperatorIDs)
		if err != nil {
			log.For(ctx).Error().Msgf("Error soft-deleting operators: %s", err.Error())
			return sr, err
		}
		if sr.HiddenOperators > 0 {
			log.For(ctx).Info().Msgf("Soft-deleted %d operators", sr.HiddenOperators)
		}
	}
	// Update any automatically generated agency-operator associations
	if err := UpdateFeedGeneratedOperators(ctx, atx, sr.FeedIDs); err != nil {
		sr.Errors = append(sr.Errors, err)
	}
	// Rollback on any errors
	if len(sr.Errors) > 0 {
		log.For(ctx).Error().Msgf("Rollback due to one or more failures")
		return sr, fmt.Errorf("failed: %s", sr.Errors[0].Error())
	}
	return sr, nil
}
