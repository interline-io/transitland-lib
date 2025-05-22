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
	TransferType int64
}

func (e *TransferStopLocationTypeError) Error() string {
	var requiredType string
	if e.TransferType == 4 || e.TransferType == 5 {
		requiredType = "0 (stop/platform)"
	} else {
		requiredType = "0 (stop/platform) or 1 (station)"
	}
	return fmt.Sprintf(
		"transfer field '%s' references stop '%s' which has location_type %d but must be %s",
		e.FieldName,
		e.StopID,
		e.LocationType,
		requiredType,
	)
}

// TransferStopLocationTypeCheck checks that stops referenced in transfers.txt have valid location_types.
// According to GTFS spec:
// - For transfer_type 0,1,2,3: stops must have location_type 0 (stop/platform) or 1 (station)
// - For transfer_type 4,5: fields are optional, but when provided must have location_type 0 (stop/platform)
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

// isValidTransferLocationType checks if the location_type is valid based on transfer_type:
// - For transfer_type 0,1,2,3: location_type must be 0 (stop/platform) or 1 (station)
// - For transfer_type 4,5: location_type must be 0 (stop/platform)
func isValidTransferLocationType(locationType int, transferType int64) bool {
	if transferType == 4 || transferType == 5 {
		return locationType == 0
	}
	return locationType == 0 || locationType == 1
}

func (e *TransferStopLocationTypeCheck) Validate(ent tt.Entity) []error {
	transfer, ok := ent.(*gtfs.Transfer)
	if !ok {
		return nil
	}

	var errs []error

	// For transfer_type 4,5: fields are optional but must be location_type=0 if provided
	// For other transfer_types: fields are required and must be location_type=0,1
	if fromType, ok := e.locationTypes[transfer.FromStopID.Val]; ok {
		if !isValidTransferLocationType(fromType, transfer.TransferType.Val) {
			errs = append(errs, &TransferStopLocationTypeError{
				StopID:       transfer.FromStopID.Val,
				FieldName:    "from_stop_id",
				LocationType: fromType,
				TransferType: transfer.TransferType.Val,
			})
		}
	}

	if toType, ok := e.locationTypes[transfer.ToStopID.Val]; ok {
		if !isValidTransferLocationType(toType, transfer.TransferType.Val) {
			errs = append(errs, &TransferStopLocationTypeError{
				StopID:       transfer.ToStopID.Val,
				FieldName:    "to_stop_id",
				LocationType: toType,
				TransferType: transfer.TransferType.Val,
			})
		}
	}

	return errs
}
