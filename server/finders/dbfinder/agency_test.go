package dbfinder

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/testutil"
	sq "github.com/irees/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The fixture agency these tests change, inside a transaction rolled back after.
const testAgencyID = "caltrain-ca-us"

// testTx opens a transaction on the test database, rolled back when the test ends.
func testTx(t *testing.T) *sqlx.Tx {
	tx, err := testutil.MustOpenTestDB(t).BeginTxx(context.Background(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })
	return tx
}

func testExec(t *testing.T, tx *sqlx.Tx, q sq.Sqlizer) {
	qstr, qargs, err := q.ToSql()
	require.NoError(t, err)
	_, err = tx.Exec(qstr, qargs...)
	require.NoError(t, err)
}

// testAgency returns the database ids of Caltrain's active agency and of its
// operator row.
func testAgency(t *testing.T, tx *sqlx.Tx) (agencyID int, coifID int) {
	ctx := context.Background()
	require.NoError(t, dbutil.Get(ctx, tx, sq.StatementBuilder.
		Select("gtfs_agencies.id").
		From("gtfs_agencies").
		Join("feed_states on feed_states.materialized_feed_version_id = gtfs_agencies.feed_version_id").
		Where(sq.Eq{"gtfs_agencies.agency_id": testAgencyID}), &agencyID))
	require.NoError(t, dbutil.Get(ctx, tx, sq.StatementBuilder.
		Select("id").
		From("current_operators_in_feed").
		Where(sq.Eq{"resolved_gtfs_agency_id": testAgencyID}), &coifID))
	return agencyID, coifID
}

// testCopyOperatorRow inserts a copy of an operator row, with or without its
// operator_id.
func testCopyOperatorRow(t *testing.T, tx *sqlx.Tx, coifID int, withOperator bool) {
	psq := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	cols := []string{"feed_id", "resolved_gtfs_agency_id", "resolved_onestop_id", "resolved_name"}
	if withOperator {
		cols = append(cols, "operator_id")
	}
	testExec(t, tx, psq.Insert("current_operators_in_feed").
		Columns(cols...).
		Select(psq.Select(cols...).From("current_operators_in_feed").Where(sq.Eq{"id": coifID})))
}

func TestFinder_FindAgencies_OneOperatorRow(t *testing.T) {
	// An agency with two operator rows is returned once. The Atlas row wins over a
	// generated one: here the Atlas row is the newer copy, and the original fixture
	// row is made generated.
	ctx := context.Background()
	tx := testTx(t)
	agencyID, coifID := testAgency(t, tx)
	testCopyOperatorRow(t, tx, coifID, true)
	testExec(t, tx, sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Update("current_operators_in_feed").Set("operator_id", nil).Where(sq.Eq{"id": coifID}))
	atlasID := 0
	require.NoError(t, dbutil.Get(ctx, tx, sq.StatementBuilder.
		Select("id").
		From("current_operators_in_feed").
		Where(sq.Eq{"resolved_gtfs_agency_id": testAgencyID}).
		Where(sq.NotEq{"operator_id": nil}), &atlasID))
	require.NotEqual(t, coifID, atlasID)

	got, err := NewFinder(tx).FindAgencies(ctx, nil, nil, []int{agencyID}, nil)
	require.NoError(t, err)
	if assert.Len(t, got, 1) && assert.NotNil(t, got[0].CoifID) {
		assert.Equal(t, atlasID, *got[0].CoifID)
		assert.Equal(t, "o-9q9-caltrain", got[0].OnestopID)
	}
}

func TestFinder_FindAgencies_NoAgencyID(t *testing.T) {
	// An agency without an agency_id, which holds '', finds its operator row when
	// that row holds NULL, as one written before its feed had a version does.
	ctx := context.Background()
	tx := testTx(t)
	agencyID, coifID := testAgency(t, tx)
	psq := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	testExec(t, tx, psq.Update("gtfs_agencies").Set("agency_id", "").Where(sq.Eq{"id": agencyID}))
	testExec(t, tx, psq.Update("current_operators_in_feed").Set("resolved_gtfs_agency_id", nil).Where(sq.Eq{"id": coifID}))

	got, err := NewFinder(tx).FindAgencies(ctx, nil, nil, []int{agencyID}, nil)
	require.NoError(t, err)
	if assert.Len(t, got, 1) {
		assert.Equal(t, "o-9q9-caltrain", got[0].OnestopID)
	}
	ops, errs := NewFinder(tx).OperatorsByAgencyIDs(ctx, []int{agencyID})
	require.Empty(t, errs)
	if assert.Len(t, ops, 1) && assert.Len(t, ops[0], 1) {
		assert.Equal(t, "o-9q9-caltrain", ops[0][0].OnestopID.Val)
	}
}
