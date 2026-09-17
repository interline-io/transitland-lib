package dbfinder

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/testutil"
	sq "github.com/irees/squirrel"
	"github.com/stretchr/testify/assert"
)

// An agency associated with two operators loads both.
func TestFinder_OperatorsByAgencyIDs(t *testing.T) {
	ctx := context.Background()
	tx, err := testutil.MustOpenTestDB(t).BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	insert := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Insert("current_operators_in_feed").
		Columns("feed_id", "resolved_gtfs_agency_id", "resolved_onestop_id", "resolved_name").
		Select(sq.StatementBuilder.
			Select("feed_id", "resolved_gtfs_agency_id").
			Column("?", "o-9q9-caltrain~second").
			Column("?", "Second").
			From("current_operators_in_feed").
			Where(sq.Eq{"resolved_gtfs_agency_id": "caltrain-ca-us"}))
	qstr, qargs, err := insert.ToSql()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, qstr, qargs...); err != nil {
		t.Fatal(err)
	}
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
