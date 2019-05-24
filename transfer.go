package gotransit

import (
	"github.com/interline-io/gotransit/causes"
)

// Transfer transfers.txt
type Transfer struct {
	FromStopID      string `csv:"from_stop_id" required:"true" gorm:"type:int;index;not null"`
	ToStopID        string `csv:"to_stop_id" required:"true" gorm:"type:int;index;not null"`
	TransferType    int    `csv:"transfer_type" required:"true" min:"0" max:"3" gorm:"index;not null"`
	MinTransferTime int    `csv:"min_transfer_time" min:"0"`
	BaseEntity
}

// EntityID returns nothing, Transfers are not unique.
func (ent *Transfer) EntityID() string {
	return ""
}

// Warnings for this Entity.
func (ent *Transfer) Warnings() (errs []error) {
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
	errs = ValidateTags(ent)
	errs = append(errs, ent.BaseEntity.loadErrors...)
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
	if fromStopID, ok := emap.Get(&Stop{StopID: ent.FromStopID}); ok {
		ent.FromStopID = fromStopID
	} else {
		return causes.NewInvalidReferenceError("from_stop_id", ent.FromStopID)
	}
	if toStopID, ok := emap.Get(&Stop{StopID: ent.ToStopID}); ok {
		ent.ToStopID = toStopID
	} else {
		return causes.NewInvalidReferenceError("to_stop_id", ent.ToStopID)
	}
	return nil
}
