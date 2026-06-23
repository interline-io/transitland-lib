package feedmanager

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/feedstate"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
)

// PostgresFeedManager implements FeedManager over a tldb.Adapter. Each method is
// a transcription of the direct adapter call it replaces in the import (and
// fetch) flow, so it is behavior-identical to those call sites. Despite the name
// it works against any tldb.Adapter (e.g. sqlite in tests).
type PostgresFeedManager struct {
	adapter tldb.Adapter
	// inTx marks a manager bound to an open transaction (see WithTx). The
	// underlying adapter.Tx is not re-entrant, so a nested WithTx must reuse it.
	inTx bool
}

var _ FeedManager = (*PostgresFeedManager)(nil)

// NewPostgresFeedManager wraps a tldb.Adapter.
func NewPostgresFeedManager(adapter tldb.Adapter) *PostgresFeedManager {
	return &PostgresFeedManager{adapter: adapter}
}

func (m *PostgresFeedManager) GetFeedVersion(ctx context.Context, fvid int) (*dmfr.FeedVersion, error) {
	fv := dmfr.FeedVersion{}
	fv.ID = fvid
	if err := m.adapter.Find(ctx, &fv); err != nil {
		return nil, err
	}
	return &fv, nil
}

func (m *PostgresFeedManager) GetFeedVersionImport(ctx context.Context, fvid int) (*dmfr.FeedVersionImport, error) {
	// Mirrors the importer's existence check (SELECT ... WHERE feed_version_id = ?
	// with ErrNoRows treated as "no import yet"), but scans the whole record.
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

func (m *PostgresFeedManager) CreateFeedVersionImport(ctx context.Context, fvi *dmfr.FeedVersionImport) (int, error) {
	id, err := m.adapter.Insert(ctx, fvi)
	if err != nil {
		return 0, err
	}
	fvi.ID = id
	return id, nil
}

func (m *PostgresFeedManager) UpdateFeedVersionImport(ctx context.Context, fvi *dmfr.FeedVersionImport) error {
	return m.adapter.Update(ctx, fvi)
}

func (m *PostgresFeedManager) ActivateFeedVersion(ctx context.Context, fvid int) error {
	// Transcribes importer.ActivateFeedVersion: delegate to the feedstate manager,
	// which replaces any existing active version and refreshes materialized tables.
	if err := feedstate.NewManager(m.adapter).ActivateFeedVersion(ctx, fvid); err != nil {
		return fmt.Errorf("failed to activate feed version: %w", err)
	}
	log.For(ctx).Info().Int("feed_version_id", fvid).Msg("Successfully activated feed version")
	return nil
}

func (m *PostgresFeedManager) EntityWriter(fvid int) adapters.Writer {
	return &tldb.Writer{Adapter: m.adapter, FeedVersionID: fvid}
}

func (m *PostgresFeedManager) OpenReader(ctx context.Context, fv *dmfr.FeedVersion, storage string) (adapters.Reader, error) {
	tladapter, err := tlcsv.NewStoreAdapter(ctx, storage, fv.File, fv.Fragment.Val)
	if err != nil {
		return nil, err
	}
	return tlcsv.NewReaderFromAdapter(tladapter)
}

func (m *PostgresFeedManager) WithTx(ctx context.Context, fn func(context.Context, FeedManager) error) error {
	if m.inTx {
		// Already inside a transaction; adapter.Tx is not re-entrant, so join it.
		return fn(ctx, m)
	}
	return m.adapter.Tx(func(atx tldb.Adapter) error {
		return fn(ctx, &PostgresFeedManager{adapter: atx, inTx: true})
	})
}
