package sync

import (
	"context"
	"database/sql"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
)

// UpdateOperator updates or inserts a single operator, as well as managing associated operator-in-feed records
func UpdateOperator(ctx context.Context, atx tldb.Adapter, operator dmfr.Operator) (int, bool, bool, error) {
	// Check if we have the existing operator
	found := false
	updated := false
	var errTx error
	ent := dmfr.Operator{}
	err := atx.Get(ctx, &ent, "SELECT * FROM current_operators WHERE onestop_id = ?", operator.OnestopID)
	if err == nil {
		// Exists, update key values
		found = true
		operator.ID = ent.ID
		if !ent.Equal(&operator) {
			updated = true
			operator.CreatedAt = ent.CreatedAt
			operator.DeletedAt = tt.Time{}
			errTx = atx.Update(ctx, &operator)
		}
	} else if err == sql.ErrNoRows {
		// Insert
		operator.ID, errTx = atx.Insert(ctx, &operator)
	} else {
		// Error
		errTx = err
	}
	if errTx != nil {
		return 0, false, false, errTx
	}
	// Update operator in feeds
	// This happens even if the entity did not change.
	oifUpdate, err := updateOifs(ctx, atx, operator)
	if err != nil {
		return 0, false, false, err
	}
	if oifUpdate {
		updated = true
	}
	return operator.ID, found, updated, nil
}

// HideUnseedOperators .
func HideUnseedOperators(atx tldb.Adapter, found []int) (int, error) {
	// Delete unreferenced feeds
	t := tt.NewTime(time.Now().UTC())
	r, err := atx.Sqrl().
		Update("current_operators").
		Where(sq.NotEq{"id": found}).
		Where(sq.Eq{"deleted_at": nil}).
		Set("deleted_at", t).
		Exec()
	if err != nil {
		return 0, err
	}
	c, err := r.RowsAffected()
	return int(c), err
}
