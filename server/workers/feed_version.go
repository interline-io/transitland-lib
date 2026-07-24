// Package workers provides reference job workers for the standalone server.
// Production deployments generally register their own workers; these exist so
// the demo server and tests can process the feed-version import/unimport jobs
// that the GraphQL mutations enqueue.
package workers

import (
	"context"

	"github.com/interline-io/transitland-lib/server/model"
)

// FeedVersionImportWorker imports a feed version on a job context.
type FeedVersionImportWorker struct {
	FeedVersionID int `json:"feed_version_id"`
}

func (w *FeedVersionImportWorker) Kind() string {
	return "feed-version-import"
}

func (w *FeedVersionImportWorker) Run(ctx context.Context) error {
	_, err := model.ForContext(ctx).Actions.FeedVersionImport(ctx, w.FeedVersionID)
	return err
}

// FeedVersionUnimportWorker unimports a feed version on a job context.
type FeedVersionUnimportWorker struct {
	FeedVersionID int `json:"feed_version_id"`
}

func (w *FeedVersionUnimportWorker) Kind() string {
	return "feed-version-unimport"
}

func (w *FeedVersionUnimportWorker) Run(ctx context.Context) error {
	_, err := model.ForContext(ctx).Actions.FeedVersionUnimport(ctx, w.FeedVersionID)
	return err
}
