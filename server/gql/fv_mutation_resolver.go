package gql

import (
	"context"
	"errors"
	"io"

	"github.com/99designs/gqlgen/graphql"

	"github.com/interline-io/transitland-lib/model"
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
	return model.ForContext(ctx).Actions.FeedVersionImport(ctx, fvid)
}

func (r *mutationResolver) FeedVersionUnimport(ctx context.Context, fvid int) (*model.FeedVersionUnimportResult, error) {
	return model.ForContext(ctx).Actions.FeedVersionUnimport(ctx, fvid)
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
