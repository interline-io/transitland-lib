package gotransit

import (
	"fmt"

	"github.com/interline-io/gotransit/causes"
)

// FareAttribute fare_attributes.txt
type FareAttribute struct {
	FareID           string               `csv:"fare_id" required:"true"`
	Price            float64              `csv:"price" required:"true" min:"0"`
	CurrencyType     string               `csv:"currency_type" required:"true" validator:"currency"`
	PaymentMethod    int                  `csv:"payment_method" required:"true" min:"0" max:"1"`
	Transfers        string               `csv:"transfers"` // string, empty is meaningful
	AgencyID         OptionalRelationship `csv:"agency_id" `
	TransferDuration int                  `csv:"transfer_duration" min:"0"`
	BaseEntity
}

// EntityID returns the ID or FareID.
func (ent *FareAttribute) EntityID() string {
	return entID(ent.ID, ent.FareID)
}

// Warnings for this Entity.
func (ent *FareAttribute) Warnings() (errs []error) {
	return errs
}

// Errors for this Entity.
func (ent *FareAttribute) Errors() (errs []error) {
	errs = ValidateTags(ent)
	errs = append(errs, ent.BaseEntity.loadErrors...)
	switch ent.Transfers {
	case "":
	case "0":
	case "1":
	case "2":
	default:
		errs = append(errs, causes.NewInvalidFieldError("transfers", ent.Transfers, fmt.Errorf("invalid transfers, must be empty, 0, 1, or 2")))
	}
	return errs
}

// Filename fare_attributes.txt
func (ent *FareAttribute) Filename() string {
	return "fare_attributes.txt"
}

// TableName gtfs_fare_attributes
func (ent *FareAttribute) TableName() string {
	return "gtfs_fare_attributes"
}

// UpdateKeys updates Entity references.
func (ent *FareAttribute) UpdateKeys(emap *EntityMap) error {
	// Adjust AgencyID - optional
	if len(ent.AgencyID.Key) > 0 {
		if agencyID, ok := emap.GetEntity(&Agency{AgencyID: ent.AgencyID.Key}); ok {
			ent.AgencyID.Key = agencyID
		} else {
			return causes.NewInvalidReferenceError("agency_id", ent.AgencyID.Key)
		}
	}
	return nil
}
