package tl

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/enum"
)

// Transfer transfers.txt
type Transfer struct {
	FromStopID      string `csv:"from_stop_id" required:"true"`
	ToStopID        string `csv:"to_stop_id" required:"true"`
	TransferType    int    `csv:"transfer_type" required:"true"`
	MinTransferTime int    `csv:"min_transfer_time"`
	BaseEntity
}

// EntityID returns nothing, Transfers are not unique.
func (ent *Transfer) EntityID() string {
	return ""
}

// Warnings for this Entity.
func (ent *Transfer) Warnings() (errs []error) {
	errs = append(errs, ent.loadWarnings...)
	if ent.TransferType != 2 && ent.MinTransferTime != 0 {
		errs = append(errs, causes.NewValidationWarning("min_transfer_time", "should not set min_transfer_time unless transfer_type = 2"))
	}
	if ent.TransferType == 2 && ent.MinTransferTime == 0 {
		errs = append(errs, causes.NewValidationWarning("min_transfer_time", "transfer_type = 2 requires min_transfer_time to be set"))
	}
	return errs
}

// Errors for this Entity.
func (ent *Transfer) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, enum.CheckPresent("from_stop_id", ent.FromStopID)...)
	errs = append(errs, enum.CheckPresent("to_stop_id", ent.ToStopID)...)
	errs = append(errs, enum.CheckInsideRangeInt("transfer_type", ent.TransferType, 0, 3)...)
	errs = append(errs, enum.CheckPositiveInt("min_transfer_time", ent.MinTransferTime)...)
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
	if fromStopID, ok := emap.GetEntity(&Stop{StopID: ent.FromStopID}); ok {
		ent.FromStopID = fromStopID
	} else {
		return causes.NewInvalidReferenceError("from_stop_id", ent.FromStopID)
	}
	if toStopID, ok := emap.GetEntity(&Stop{StopID: ent.ToStopID}); ok {
		ent.ToStopID = toStopID
	} else {
		return causes.NewInvalidReferenceError("to_stop_id", ent.ToStopID)
	}
	return nil
}
