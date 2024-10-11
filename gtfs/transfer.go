package gtfs

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// Transfer transfers.txt
type Transfer struct {
	FromStopID      string
	ToStopID        string
	FromRouteID     tt.Key
	ToRouteID       tt.Key
	FromTripID      tt.Key
	ToTripID        tt.Key
	TransferType    int
	MinTransferTime tt.Int
	tt.BaseEntity
}

// Errors for this Entity.
func (ent *Transfer) Errors() (errs []error) {
	// transfer_type is required but can also be empty, so hard to distinguish
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, tt.CheckInsideRangeInt("transfer_type", ent.TransferType, 0, 5)...)
	errs = append(errs, tt.CheckPositiveInt("min_transfer_time", ent.MinTransferTime.Val)...)
	// FromStopID, ToStopID required
	errs = append(errs, tt.CheckPresent("from_stop_id", ent.FromStopID)...)
	errs = append(errs, tt.CheckPresent("to_stop_id", ent.ToStopID)...)
	if ent.TransferType == 4 || ent.TransferType == 5 {
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

// UpdateKeys updates entity references.
func (ent *Transfer) UpdateKeys(emap *EntityMap) error {
	// Adjust StopIDs
	if ent.FromStopID != "" {
		if fromStopID, ok := emap.GetEntity(&Stop{StopID: ent.FromStopID}); ok {
			ent.FromStopID = fromStopID
		} else {
			return causes.NewInvalidReferenceError("from_stop_id", ent.FromStopID)
		}
	}
	if ent.ToStopID != "" {
		if toStopID, ok := emap.GetEntity(&Stop{StopID: ent.ToStopID}); ok {
			ent.ToStopID = toStopID
		} else {
			return causes.NewInvalidReferenceError("to_stop_id", ent.ToStopID)
		}
	}
	// Adjust RouteIDs
	if ent.FromRouteID.Valid {
		if fromRouteID, ok := emap.GetEntity(&Route{RouteID: ent.FromRouteID.Val}); ok {
			ent.FromRouteID = tt.NewKey(fromRouteID)
		} else {
			return causes.NewInvalidReferenceError("from_route_id", ent.FromRouteID.Val)
		}
	}
	if ent.ToRouteID.Valid {
		if toRouteID, ok := emap.GetEntity(&Route{RouteID: ent.ToRouteID.Val}); ok {
			ent.ToRouteID = tt.NewKey(toRouteID)
		} else {
			return causes.NewInvalidReferenceError("to_route_id", ent.ToRouteID.Val)
		}
	}
	// Adjust TripIDs
	if ent.FromTripID.Valid {
		if fromTripID, ok := emap.GetEntity(&Trip{TripID: ent.FromTripID.Val}); ok {
			ent.FromTripID = tt.NewKey(fromTripID)
		} else {
			return causes.NewInvalidReferenceError("from_trip_id", ent.FromTripID.Val)
		}
	}
	if ent.ToTripID.Valid {
		if toTripID, ok := emap.GetEntity(&Trip{TripID: ent.ToTripID.Val}); ok {
			ent.ToTripID = tt.NewKey(toTripID)
		} else {
			return causes.NewInvalidReferenceError("to_trip_id", ent.ToTripID.Val)
		}
	}
	return nil
}
