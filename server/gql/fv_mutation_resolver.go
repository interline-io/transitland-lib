package gql

import (
	"context"
	"errors"
	"io"

	"github.com/99designs/gqlgen/graphql"

	"github.com/interline-io/transitland-lib/server/auth/authz"
	"github.com/interline-io/transitland-lib/server/jobs"
	"github.com/interline-io/transitland-lib/server/model"
)

// Job kinds the import/unimport mutations enqueue. These match the Kind() of
// the workers that process them (server/workers here, private workers in the
// deployment binary); the string is the queue contract, so the resolver stays
// free of a worker-package dependency.
const (
	feedVersionImportJobKind   = "feed-version-import"
	feedVersionUnimportJobKind = "feed-version-unimport"
)

// mutation root

type mutationResolver struct{ *Resolver }

func (r *mutationResolver) ValidateGtfs(ctx context.Context, file *graphql.Upload, url *string, rturls []string) (*model.ValidationReport, error) {
	var src io.Reader
	if file != nil {
		src = file.File
	}
	return model.ForContext(ctx).Actions.ValidateUpload(ctx, src, url, rturls)
}

func (r *mutationResolver) FeedVersionFetch(ctx context.Context, file *graphql.Upload, url *string, feedId string) (*model.FeedVersionFetchResult, error) {
	var feedSrc io.Reader
	if file != nil {
		feedSrc = file.File
	}
	feedUrl := ""
	if url != nil {
		feedUrl = *url
	}
	return model.ForContext(ctx).Actions.StaticFetch(ctx, feedId, feedSrc, feedUrl)
}

func (r *mutationResolver) FeedVersionImport(ctx context.Context, fvid int) (*model.FeedVersionImportResult, error) {
	cfg := model.ForContext(ctx)
	if cfg.Jobs == nil {
		return cfg.Actions.FeedVersionImport(ctx, fvid)
	}
	if err := checkFeedVersionEdit(ctx, fvid); err != nil {
		return nil, err
	}
	success, err := enqueueAndWaitFeedVersion(ctx, feedVersionImportJobKind, fvid)
	if err != nil {
		return nil, err
	}
	return &model.FeedVersionImportResult{Success: success}, nil
}

func (r *mutationResolver) FeedVersionUnimport(ctx context.Context, fvid int) (*model.FeedVersionUnimportResult, error) {
	cfg := model.ForContext(ctx)
	if cfg.Jobs == nil {
		return cfg.Actions.FeedVersionUnimport(ctx, fvid)
	}
	if err := checkFeedVersionEdit(ctx, fvid); err != nil {
		return nil, err
	}
	success, err := enqueueAndWaitFeedVersion(ctx, feedVersionUnimportJobKind, fvid)
	if err != nil {
		return nil, err
	}
	return &model.FeedVersionUnimportResult{Success: success}, nil
}

// checkFeedVersionEdit gates the import/unimport mutations on the request
// context. The enqueued worker re-runs the action (and its own check) on a job
// context, so this preserves the caller's CanEdit check on the specific feed
// version rather than deferring authorization to the worker's identity.
func checkFeedVersionEdit(ctx context.Context, fvid int) error {
	if fvid <= 0 {
		return errors.New("invalid feed version id")
	}
	cfg := model.ForContext(ctx)
	if cfg.Checker == nil {
		return authz.ErrUnauthorized
	}
	ok, err := cfg.Checker.Check(ctx, authz.ObjectRef{Type: authz.FeedVersionType, ID: int64(fvid)}, authz.CanEdit)
	if err != nil {
		return err
	}
	if !ok {
		return authz.ErrUnauthorized
	}
	return nil
}

// enqueueAndWaitFeedVersion submits a feed-version job and blocks until it
// reaches a terminal state, so the mutation keeps its synchronous result while
// the work runs on a job context that survives request cancellation. A queue
// that can't report status (no StatusQueue) is treated as fire-and-forget.
func enqueueAndWaitFeedVersion(ctx context.Context, kind string, fvid int) (bool, error) {
	q, err := model.ForContext(ctx).Jobs.Queue(kind)
	if err != nil {
		return false, err
	}
	st, err := q.Submit(ctx, jobs.Job{Kind: kind, Args: jobs.Args{"feed_version_id": fvid}})
	if err != nil {
		return false, err
	}
	sq, ok := q.(jobs.StatusQueue)
	if !ok {
		return true, nil
	}
	ch, err := sq.Watch(ctx, st.Job.ID)
	if err != nil {
		return false, err
	}
	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case ev, open := <-ch:
			if !open {
				return false, errors.New("job watch ended without terminal state for job")
			}
			switch ev.State {
			case jobs.JobStateSucceeded:
				return true, nil
			case jobs.JobStateFailed, jobs.JobStateCancelled:
				if ev.Message != "" {
					return false, errors.New(ev.Message)
				}
				return false, nil
			}
		}
	}
}

func (r *mutationResolver) FeedVersionUpdate(ctx context.Context, values model.FeedVersionSetInput) (*model.FeedVersion, error) {
	cfg := model.ForContext(ctx)
	entId, err := cfg.Actions.FeedVersionUpdate(ctx, values)
	if err != nil {
		return nil, err
	}
	ents, errs := cfg.Finder.FeedVersionsByIDs(ctx, []int{entId})
	return first(errs, ents)
}

func (r *mutationResolver) FeedVersionDelete(ctx context.Context, id int) (*model.FeedVersionDeleteResult, error) {
	return nil, errors.New("temporarily unavailable")
}
