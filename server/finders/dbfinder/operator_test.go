package dbfinder

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/sync"
	"github.com/interline-io/transitland-lib/tldb/postgres"
	"github.com/interline-io/transitland-lib/tt"
	sq "github.com/irees/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

// addSecondOperator syncs a second Atlas operator, o-9q9-caltrain~second, that
// claims Caltrain's agency.
func addSecondOperator(ctx context.Context, t *testing.T, tx *sqlx.Tx) {
	t.Helper()
	atx := postgres.NewPostgresAdapterFromDBX(tx)
	agencies, err := sync.ActiveFeedAgencies(ctx, atx)
	if err != nil {
		t.Fatal(err)
	}
	op := dmfr.Operator{
		OnestopID:       tt.NewString("o-9q9-caltrain~second"),
		Name:            tt.NewString("Second"),
		File:            tt.NewString("test.dmfr.json"),
		AssociatedFeeds: []dmfr.OperatorAssociatedFeed{{FeedOnestopID: tt.NewString("CT")}},
	}
	if _, _, _, err := sync.UpdateOperator(ctx, atx, op, agencies); err != nil {
		t.Fatal(err)
	}
}

// caltrainRows returns the rows of Caltrain's operator, one for each of its
// feeds, in id order.
func caltrainRows(ctx context.Context, t *testing.T, f *Finder) []*model.Operator {
	t.Helper()
	rows, err := f.FindOperators(ctx, nil, nil, nil, &model.OperatorFilter{OnestopID: ptr("o-9q9-caltrain"), Merged: ptr(false)})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 2 {
		t.Fatalf("o-9q9-caltrain has %d rows, want at least 2", len(rows))
	}
	return rows
}

// An agency associated with two operators loads both.
func TestFinder_OperatorsByAgencyIDs(t *testing.T) {
	ctx := context.Background()
	tx, err := testutil.MustOpenTestDB(t).BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	addSecondOperator(ctx, t, tx)
	var agencyID int
	agency := sq.StatementBuilder.
		Select("gtfs_agencies.id").
		From("gtfs_agencies").
		Join("feed_states on feed_states.materialized_feed_version_id = gtfs_agencies.feed_version_id").
		Where(sq.Eq{"gtfs_agencies.agency_id": "caltrain-ca-us"})
	if err := dbutil.Get(ctx, tx, agency, &agencyID); err != nil {
		t.Fatal(err)
	}

	got, errs := NewFinder(tx).OperatorsByAgencyIDs(ctx, []int{agencyID})
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	var osids []string
	if assert.Len(t, got, 1) {
		for _, op := range got[0] {
			osids = append(osids, op.OnestopID.Val)
		}
	}
	assert.Equal(t, []string{"o-9q9-caltrain", "o-9q9-caltrain~second"}, osids)
}

// Rows that share a onestop id each load their own row when asked for together.
func TestFinder_OperatorsByCOIFs(t *testing.T) {
	ctx := context.Background()
	f := NewFinder(testutil.MustOpenTestDB(t))
	rows := caltrainRows(ctx, t, f)
	var ids []int
	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	got, errs := f.OperatorsByCOIFs(ctx, ids)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	assert.Equal(t, rows, got)
}

// Rows of one operator merge to the row with the lowest id, in whatever order a
// scan reads them.
func TestFinder_FindOperators_MergedRow(t *testing.T) {
	ctx := context.Background()
	tx, err := testutil.MustOpenTestDB(t).BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	f := NewFinder(tx)
	first := caltrainRows(ctx, t, f)[0].ID
	// An update writes a row anew, after the others, so a scan reads it last, as it
	// reads a row that sync has refreshed.
	if _, err := tx.ExecContext(ctx, "update current_operators_in_feed set resolved_name = resolved_name where id = $1", first); err != nil {
		t.Fatal(err)
	}

	got, err := f.FindOperators(ctx, nil, nil, nil, &model.OperatorFilter{OnestopID: ptr("o-9q9-caltrain")})
	if err != nil {
		t.Fatal(err)
	}
	if assert.Len(t, got, 1) {
		assert.Equal(t, first, got[0].ID)
	}
}
