package rules

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// MinTransferTimeCheck reports when min_transfer_type is required to be set
type MinTransferTimeCheck struct{}

// Validate .
func (e *MinTransferTimeCheck) Validate(ent tt.Entity) []error {
	if v, ok := ent.(*gtfs.Transfer); ok {
		var errs []error
		if v.TransferType != 2 && v.MinTransferTime.Valid {
			errs = append(errs, causes.NewValidationWarning("min_transfer_time", "should not set min_transfer_time unless transfer_type = 2"))
		}
		if v.TransferType == 2 && !v.MinTransferTime.Valid {
			errs = append(errs, causes.NewValidationWarning("min_transfer_time", "transfer_type = 2 requires min_transfer_time to be set"))
		}
		return errs
	}
	return nil

}
