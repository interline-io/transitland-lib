package actions

import (
	"context"
	"io"

	"github.com/interline-io/transitland-lib/server/model"
)

func init() {
	var _ model.Actions = &Actions{}
}

type Actions struct{}

func (Actions) StaticFetch(ctx context.Context, feedId string, feedSrc io.Reader, feedUrl string) (*model.FeedVersionFetchResult, error) {
	return StaticFetch(ctx, feedId, feedSrc, feedUrl)
}

func (Actions) RTFetch(ctx context.Context, target string, feedId string, feedUrl string, urlType string) error {
	return RTFetch(ctx, target, feedId, feedUrl, urlType)
}

func (Actions) ValidateUpload(ctx context.Context, src io.Reader, feedURL *string, rturls []string) (*model.ValidationReport, error) {
	return ValidateUpload(ctx, src, feedURL, rturls)
}

func (Actions) GbfsFetch(ctx context.Context, feedId string, feedUrl string) error {
	return GbfsFetch(ctx, feedId, feedUrl)
}

func (Actions) FeedVersionImport(ctx context.Context, fvid int) (*model.FeedVersionImportResult, error) {
	return FeedVersionImport(ctx, fvid)
}

func (Actions) FeedVersionUnimport(ctx context.Context, fvid int) (*model.FeedVersionUnimportResult, error) {
	return FeedVersionUnimport(ctx, fvid)
}

func (Actions) FeedVersionUpdate(ctx context.Context, values model.FeedVersionSetInput) (int, error) {
	return FeedVersionUpdate(ctx, values)
}

func (Actions) FeedVersionDelete(ctx context.Context, fvid int) (*model.FeedVersionDeleteResult, error) {
	return FeedVersionDelete(ctx, fvid)
}
