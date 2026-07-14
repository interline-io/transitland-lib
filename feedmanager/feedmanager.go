// Package feedmanager defines the bookkeeping the import and fetch flows need
// beyond the Copier's entity writes (adapters.Writer): feed-version
// dedup/creation, the feed_version_gtfs_import lifecycle, and activation.
//
// It is deliberately domain-shaped, not SQL-shaped — no Sqrl/Get/Select leaks
// through, and entity writes stay on adapters.Writer.
package feedmanager

import (
	"context"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/stats"
	"github.com/interline-io/transitland-lib/validator"
)

// FeedManager is the metadata-bookkeeping surface for the import and fetch flows.
type FeedManager interface {
	GetFeedVersion(ctx context.Context, fvid int) (*dmfr.FeedVersion, error)

	// Returns (nil, nil) when no import exists yet — absence is not an error.
	GetFeedVersionImport(ctx context.Context, fvid int) (*dmfr.FeedVersionImport, error)

	// Inserts the import record and sets its new id on fvi.
	CreateFeedVersionImport(ctx context.Context, fvi *dmfr.FeedVersionImport) (int, error)

	// Writes back an existing import record's result counts / success / log.
	UpdateFeedVersionImport(ctx context.Context, fvi *dmfr.FeedVersionImport) error

	// Marks the feed version active.
	ActivateFeedVersion(ctx context.Context, fvid int) error

	// Fetch flow (in addition to the import methods above):

	// Returns the feed by id — identity + authorization.
	GetFeed(ctx context.Context, feedID int) (*dmfr.Feed, error)

	// Finds a feed version by content hash for fetch dedup; (nil, nil) when none
	// matches.
	GetFeedVersionBySHA1(ctx context.Context, sha1, sha1dir string) (*dmfr.FeedVersion, error)

	// Creates a feed version and sets its new id on fv.
	CreateFeedVersion(ctx context.Context, fv *dmfr.FeedVersion) (int, error)

	// Records a fetch attempt (response metadata, success).
	CreateFeedFetch(ctx context.Context, ff *dmfr.FeedFetch) error

	// Persists the computed feed-version stats (service levels/window, file infos,
	// onestop ids).
	WriteFeedVersionStats(ctx context.Context, fvid int, fvstats stats.FeedVersionStats) error

	// Persists a validation report.
	SaveValidationReport(ctx context.Context, fvid int, result *validator.Result, reportStorage string) error

	// The sink the Copier writes a feed version's GTFS entities to.
	EntityWriter(fvid int) adapters.Writer

	// The Copier's source for a feed version's GTFS data, returned unopened.
	OpenReader(ctx context.Context, fv *dmfr.FeedVersion, storage string) (adapters.Reader, error)

	// Runs fn in a single transaction; a nested WithTx joins the open one rather than
	// starting another. The import's entity writes are not covered -- they commit as they go.
	WithTx(ctx context.Context, fn func(ctx context.Context, tx FeedManager) error) error
}
