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
	TransferType    tt.Int `enum:"0,1,2,3,4,5"`
	MinTransferTime tt.Int `range:"0,"`
	tt.BaseEntity
}

// Filename transfers.txt
func (ent *Transfer) Filename() string {
	return "transfers.txt"
}

// TableName gtfs_transfers
func (ent *Transfer) TableName() string {
	return "gtfs_transfers"
}

// Errors for this Entity.
func (ent *Transfer) ConditionalErrors() (errs []error) {
	if ent.TransferType.Val == 1 || ent.TransferType.Val == 2 || ent.TransferType.Val == 3 {
		// FromStopID, ToStopID conditionally required
		errs = append(errs, tt.CheckConditionallyRequired("from_stop_id", ent.FromStopID.Val)...)
		errs = append(errs, tt.CheckConditionallyRequired("to_stop_id", ent.ToStopID.Val)...)
	}
	if ent.TransferType.Val == 4 || ent.TransferType.Val == 5 {
		// FromTripID, ToTripID conditionally required
		errs = append(errs, tt.CheckConditionallyRequired("from_trip_id", ent.FromTripID.Val)...)
		errs = append(errs, tt.CheckConditionallyRequired("to_trip_id", ent.ToTripID.Val)...)
	}
	if ent.TransferType.Val == 2 {
		if !ent.MinTransferTime.Valid {
			errs = append(errs, tt.CheckConditionallyRequired("min_transfer_time", "")...)
		}
	}
	return errs
}
