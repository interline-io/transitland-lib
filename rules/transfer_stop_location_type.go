package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// TransferStopLocationTypeError reports when a transfer references a stop with an invalid location_type.
type TransferStopLocationTypeError struct {
	StopID       string
	FieldName    string // from_stop_id or to_stop_id
	LocationType int
	bc
}

func (e *TransferStopLocationTypeError) Error() string {
	return fmt.Sprintf(
		"transfer field '%s' references stop '%s' which has location_type %d but must be 0 (stop/platform) or 1 (station)",
		e.FieldName,
		e.StopID,
		e.LocationType,
	)
}

// TransferStopLocationTypeCheck checks for TransferStopLocationTypeErrors.
type TransferStopLocationTypeCheck struct {
	locationTypes map[string]int
}

func (e *TransferStopLocationTypeCheck) AfterWrite(eid string, ent tt.Entity, emap *tt.EntityMap) error {
	if e.locationTypes == nil {
		e.locationTypes = map[string]int{}
	}
	if stop, ok := ent.(*gtfs.Stop); ok {
		e.locationTypes[eid] = stop.LocationType.Int()
	}
	return nil
}

// isValidTransferLocationType checks if the location_type is valid for transfers (0=stop/platform or 1=station)
func isValidTransferLocationType(locationType int) bool {
	return locationType == 0 || locationType == 1
}

func (e *TransferStopLocationTypeCheck) Validate(ent tt.Entity) []error {
	transfer, ok := ent.(*gtfs.Transfer)
	if !ok {
		return nil
	}

	var errs []error

	// Only check if transfer_type requires stop IDs
	if transfer.TransferType.Val == 1 || transfer.TransferType.Val == 2 || transfer.TransferType.Val == 3 {
		// Check from_stop_id location_type
		if fromType, ok := e.locationTypes[transfer.FromStopID.Val]; ok {
			if !isValidTransferLocationType(fromType) {
				errs = append(errs, &TransferStopLocationTypeError{
					StopID:       transfer.FromStopID.Val,
					FieldName:    "from_stop_id",
					LocationType: fromType,
				})
			}
		}

		// Check to_stop_id location_type
		if toType, ok := e.locationTypes[transfer.ToStopID.Val]; ok {
			if !isValidTransferLocationType(toType) {
				errs = append(errs, &TransferStopLocationTypeError{
					StopID:       transfer.ToStopID.Val,
					FieldName:    "to_stop_id",
					LocationType: toType,
				})
			}
		}
	}

	return errs
}
