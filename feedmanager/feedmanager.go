// Package feedmanager covers the bookkeeping the import and fetch flows need
// beyond the Copier's entity writes (adapters.Writer): feed_version
// dedup/creation, the feed_version_gtfs_import lifecycle, and activation.
//
// It's an interface so this bookkeeping can run against either a SQL database
// (DBFeedManager) or an in-memory backend — the latter lets the real import flow
// run under js/wasm with no database at all. Deliberately domain-shaped, not
// SQL-shaped: no Sqrl/Get/Select leaks through, and entity writes stay on
// adapters.Writer.
package feedmanager

import (
	"context"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/stats"
	"github.com/interline-io/transitland-lib/validator"
)

// FeedManager is the metadata-bookkeeping surface for the import and fetch
// flows. Mutating methods run inside WithTx when they must commit atomically
// with the Copier's entity writes.
type FeedManager interface {
	GetFeedVersion(ctx context.Context, fvid int) (*dmfr.FeedVersion, error)

	// Returns (nil, nil) when no import exists yet — absence is not an error.
	GetFeedVersionImport(ctx context.Context, fvid int) (*dmfr.FeedVersionImport, error)

	// Inserts the import record and sets its new id on fvi.
	CreateFeedVersionImport(ctx context.Context, fvi *dmfr.FeedVersionImport) (int, error)

	// Writes back an existing import record's result counts / success / log.
	UpdateFeedVersionImport(ctx context.Context, fvi *dmfr.FeedVersionImport) error

	// Marks the version active and refreshes any derived materialized state.
	ActivateFeedVersion(ctx context.Context, fvid int) error

	// Fetch flow (in addition to the import methods above):

	// GetFeed loads a current_feeds row by id — feed identity + authorization.
	GetFeed(ctx context.Context, feedID int) (*dmfr.Feed, error)

	// GetFeedVersionBySHA1 finds a feed version by content hash for fetch dedup;
	// (nil, nil) when none matches.
	GetFeedVersionBySHA1(ctx context.Context, sha1, sha1dir string) (*dmfr.FeedVersion, error)

	// CreateFeedVersion inserts a feed_version and sets its new id on fv.
	CreateFeedVersion(ctx context.Context, fv *dmfr.FeedVersion) (int, error)

	// CreateFeedFetch records a feed_fetch attempt (response metadata, success).
	CreateFeedFetch(ctx context.Context, ff *dmfr.FeedFetch) error

	// WriteFeedVersionStats persists the computed per-feed-version stats — service
	// levels/windows, file infos, onestop ids, geohash cells.
	WriteFeedVersionStats(ctx context.Context, fvid int, fvstats stats.FeedVersionStats) error

	// SaveValidationReport persists a validation report and its child rows.
	SaveValidationReport(ctx context.Context, fvid int, result *validator.Result, reportStorage string) error

	// EntityWriter is the Copier's entity sink. Taken from the tx-bound manager
	// inside WithTx on a SQL backend, so writes commit atomically with the import
	// metadata; in memory it's the store.
	EntityWriter(fvid int) adapters.Writer

	// OpenReader is the Copier's source, returned unopened. It hides where a
	// feed's bytes live — storage+File/Fragment on SQL, the loaded reader in
	// memory — so the import flow doesn't have to know.
	OpenReader(ctx context.Context, fv *dmfr.FeedVersion, storage string) (adapters.Reader, error)

	// WithTx runs fn in one transaction; a nested WithTx joins the open one (the
	// adapter isn't re-entrant), so the whole import commits or rolls back together.
	WithTx(ctx context.Context, fn func(ctx context.Context, tx FeedManager) error) error
}
