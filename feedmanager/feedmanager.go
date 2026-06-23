// Package feedmanager defines a narrow, semantic interface for the feed /
// feed-version / import metadata bookkeeping that the fetch and import flows
// perform — the operations that the Copier's entity writes (adapters.Writer) do
// NOT cover (feed_version dedup/creation, the feed_version_gtfs_import lifecycle,
// activation).
//
// It exists so that bookkeeping can target more than one backend. PostgresFeedManager
// wraps a tldb.Adapter and is behavior-identical to the direct adapter calls it
// replaces; an in-memory implementation (in tlv2/memfinder) can satisfy the same
// contract without a SQL database, letting the real import flow produce real
// derived metadata under js/wasm.
//
// The interface is domain-shaped, not SQL-shaped: no Sqrl/Get/Select leak
// through, and GTFS entity writes stay on adapters.Writer.
package feedmanager

import (
	"context"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/dmfr"
)

// FeedManager is the metadata-bookkeeping surface for the import (and, later,
// fetch) flows. Methods that mutate are expected to run inside WithTx when
// atomicity with the Copier's entity writes is required.
type FeedManager interface {
	// GetFeedVersion loads a feed_version by id.
	GetFeedVersion(ctx context.Context, fvid int) (*dmfr.FeedVersion, error)

	// GetFeedVersionImport returns the import record for a feed version, or
	// (nil, nil) if none exists — absence is not an error.
	GetFeedVersionImport(ctx context.Context, fvid int) (*dmfr.FeedVersionImport, error)

	// CreateFeedVersionImport inserts an import record, sets its new id on fvi, and
	// returns the id.
	CreateFeedVersionImport(ctx context.Context, fvi *dmfr.FeedVersionImport) (int, error)

	// UpdateFeedVersionImport writes an existing import record (result counts /
	// success / exception log).
	UpdateFeedVersionImport(ctx context.Context, fvi *dmfr.FeedVersionImport) error

	// ActivateFeedVersion marks a feed version active and refreshes any derived
	// materialized state.
	ActivateFeedVersion(ctx context.Context, fvid int) error

	// EntityWriter returns the adapters.Writer the Copier should write a feed
	// version's GTFS entities to. On a SQL backend (obtained from the tx-bound
	// manager inside WithTx) this is a transaction-bound writer, so entity writes
	// commit atomically with the import's metadata; the in-memory backend returns
	// its store.
	EntityWriter(fvid int) adapters.Writer

	// OpenReader returns an unopened adapters.Reader for a feed version's GTFS
	// data — the source the Copier reads. The SQL backend builds it from storage
	// (storage + fv.File/Fragment); the in-memory backend returns the reader the
	// feed was loaded from. This is the read-side symmetry of EntityWriter, so the
	// import flow never has to know where a feed's bytes live.
	OpenReader(ctx context.Context, fv *dmfr.FeedVersion, storage string) (adapters.Reader, error)

	// WithTx runs fn against a FeedManager bound to a single transaction. On a
	// SQL backend this is a real transaction; a nested WithTx joins the existing
	// one rather than opening another (the underlying adapter is not re-entrant).
	WithTx(ctx context.Context, fn func(ctx context.Context, tx FeedManager) error) error
}
