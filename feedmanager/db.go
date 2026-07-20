package feedmanager

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/feedstate"
	"github.com/interline-io/transitland-lib/stats"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/validator"
)

// DBFeedManager implements FeedManager over a tldb.Adapter — Postgres in
// production, sqlite in tests. Each method is a transcription of the direct
// adapter call it replaces in the import (and fetch) flow, so it is
// behavior-identical to those call sites.
type DBFeedManager struct {
	adapter tldb.Adapter
	// inTx marks a manager bound to an open transaction (see WithTx). The
	// underlying adapter.Tx is not re-entrant, so a nested WithTx must reuse it.
	inTx bool
}

var _ FeedManager = (*DBFeedManager)(nil)

// NewDBFeedManager wraps a tldb.Adapter.
func NewDBFeedManager(adapter tldb.Adapter) *DBFeedManager {
	return &DBFeedManager{adapter: adapter}
}

func (m *DBFeedManager) GetFeedVersion(ctx context.Context, fvid int) (*dmfr.FeedVersion, error) {
	fv := dmfr.FeedVersion{}
	fv.ID = fvid
	if err := m.adapter.Find(ctx, &fv); err != nil {
		return nil, err
	}
	return &fv, nil
}

func (m *DBFeedManager) GetFeedVersionImport(ctx context.Context, fvid int) (*dmfr.FeedVersionImport, error) {
	// ErrNoRows means no import yet, which is not a failure.
	fvi := dmfr.FeedVersionImport{}
	err := m.adapter.Get(ctx, &fvi, `SELECT * FROM feed_version_gtfs_imports WHERE feed_version_id = ?`, fvid)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &fvi, nil
}

func (m *DBFeedManager) CreateFeedVersionImport(ctx context.Context, fvi *dmfr.FeedVersionImport) (int, error) {
	id, err := m.adapter.Insert(ctx, fvi)
	if err != nil {
		return 0, err
	}
	fvi.ID = id
	return id, nil
}

func (m *DBFeedManager) UpdateFeedVersionImport(ctx context.Context, fvi *dmfr.FeedVersionImport) error {
	return m.adapter.Update(ctx, fvi)
}

func (m *DBFeedManager) ActivateFeedVersion(ctx context.Context, fvid int) error {
	// feedstate replaces any existing active version and refreshes materialized tables.
	if err := feedstate.NewManager(m.adapter).ActivateFeedVersion(ctx, fvid); err != nil {
		return fmt.Errorf("failed to activate feed version: %w", err)
	}
	log.For(ctx).Info().Int("feed_version_id", fvid).Msg("Successfully activated feed version")
	return nil
}

func (m *DBFeedManager) GetFeed(ctx context.Context, feedID int) (*dmfr.Feed, error) {
	feed := dmfr.Feed{}
	if err := m.adapter.Get(ctx, &feed, "select * from current_feeds where id = ?", feedID); err != nil {
		return nil, err
	}
	return &feed, nil
}

func (m *DBFeedManager) GetFeedVersionBySHA1(ctx context.Context, sha1, sha1dir string) (*dmfr.FeedVersion, error) {
	fv := dmfr.FeedVersion{}
	err := m.adapter.Get(ctx, &fv, "SELECT * FROM feed_versions WHERE sha1 = ? OR sha1_dir = ? LIMIT 1", sha1, sha1dir)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &fv, nil
}

func (m *DBFeedManager) CreateFeedVersion(ctx context.Context, fv *dmfr.FeedVersion) (int, error) {
	id, err := m.adapter.Insert(ctx, fv)
	if err != nil {
		return 0, err
	}
	fv.ID = id
	return id, nil
}

func (m *DBFeedManager) CreateFeedFetch(ctx context.Context, ff *dmfr.FeedFetch) error {
	id, err := m.adapter.Insert(ctx, ff)
	if err != nil {
		return err
	}
	ff.ID = id
	return nil
}

func (m *DBFeedManager) WriteFeedVersionStats(ctx context.Context, fvid int, fvstats stats.FeedVersionStats) error {
	// Honor the feed's onestop_id retention policy: stats are always built, but for a
	// version past its feed's window we drop the onestop_id data so it is not written.
	retained, err := stats.OnestopIDsRetainedForFeedVersion(ctx, m.adapter, fvid)
	if err != nil {
		return err
	}
	if !retained {
		fvstats.AgencyOnestopIDs = nil
		fvstats.RouteOnestopIDs = nil
		fvstats.StopOnestopIDs = nil
	}
	return stats.WriteFeedVersionStats(ctx, m.adapter, fvstats, fvid, stats.WriteOptions{})
}

func (m *DBFeedManager) SaveValidationReport(ctx context.Context, fvid int, result *validator.Result, reportStorage string) error {
	return validator.SaveValidationReport(ctx, m.adapter, result, fvid, reportStorage)
}

func (m *DBFeedManager) EntityWriter(fvid int) adapters.Writer {
	return &tldb.Writer{Adapter: m.adapter, FeedVersionID: fvid}
}

func (m *DBFeedManager) OpenReader(ctx context.Context, fv *dmfr.FeedVersion, storage string) (adapters.Reader, error) {
	tladapter, err := tlcsv.NewStoreAdapter(ctx, storage, fv.File, fv.Fragment.Val)
	if err != nil {
		return nil, err
	}
	return tlcsv.NewReaderFromAdapter(tladapter)
}

func (m *DBFeedManager) WithTx(ctx context.Context, fn func(context.Context, FeedManager) error) error {
	if m.inTx {
		// Already inside a transaction; adapter.Tx is not re-entrant, so join it.
		return fn(ctx, m)
	}
	return m.adapter.Tx(func(atx tldb.Adapter) error {
		return fn(ctx, &DBFeedManager{adapter: atx, inTx: true})
	})
}
