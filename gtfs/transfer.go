package gtfs

import (
	"github.com/interline-io/transitland-lib/tt"
)

// Transfer transfers.txt
type Transfer struct {
	FromStopID      tt.Key `target:"stops.txt"`
	ToStopID        tt.Key `target:"stops.txt"`
	FromRouteID     tt.Key `target:"routes.txt"`
	ToRouteID       tt.Key `target:"routes.txt"`
	FromTripID      tt.Key `target:"trips.txt"`
	ToTripID        tt.Key `target:"trips.txt"`
	TransferType    tt.Int
	MinTransferTime tt.Int
	tt.BaseEntity
}

// Errors for this Entity.
func (ent *Transfer) Errors() (errs []error) {
	// transfer_type is required but can also be empty, so hard to distinguish
	errs = append(errs, tt.CheckInsideRangeInt("transfer_type", ent.TransferType.Val, 0, 5)...)
	errs = append(errs, tt.CheckPositiveInt("min_transfer_time", ent.MinTransferTime.Val)...)
	// FromStopID, ToStopID required
	errs = append(errs, tt.CheckPresent("from_stop_id", ent.FromStopID.Val)...)
	errs = append(errs, tt.CheckPresent("to_stop_id", ent.ToStopID.Val)...)
	if ent.TransferType.Val == 4 || ent.TransferType.Val == 5 {
		// FromTripID, ToTripID conditionally required
		errs = append(errs, tt.CheckPresent("from_trip_id", ent.FromTripID.Val)...)
		errs = append(errs, tt.CheckPresent("to_trip_id", ent.ToTripID.Val)...)
	}
	return errs
}

// Filename transfers.txt
func (ent *Transfer) Filename() string {
	return "transfers.txt"
}

// TableName gtfs_transfers
func (ent *Transfer) TableName() string {
	return "gtfs_transfers"
}
