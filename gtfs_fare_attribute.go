package tl

import (
	"fmt"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/enums"
)

// FareAttribute fare_attributes.txt
type FareAttribute struct {
	FareID           string               `csv:"fare_id" required:"true"`
	Price            float64              `csv:"price" required:"true"`
	CurrencyType     string               `csv:"currency_type" required:"true"`
	PaymentMethod    int                  `csv:"payment_method" required:"true"`
	Transfers        string               `csv:"transfers"` // string, empty is meaningful
	AgencyID         OptionalRelationship `csv:"agency_id" `
	TransferDuration int                  `csv:"transfer_duration"`
	BaseEntity
}

// EntityID returns the ID or FareID.
func (ent *FareAttribute) EntityID() string {
	return entID(ent.ID, ent.FareID)
}

// Errors for this Entity.
func (ent *FareAttribute) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, enums.CheckPresent("fare_id", ent.FareID)...)
	errs = append(errs, enums.CheckPresent("currency_type", ent.CurrencyType)...)
	errs = append(errs, enums.CheckPositive("price", ent.Price)...)
	errs = append(errs, enums.CheckCurrency("currency_type", ent.CurrencyType)...)
	errs = append(errs, enums.CheckInsideRangeInt("payment_method", ent.PaymentMethod, 0, 1)...)
	errs = append(errs, enums.CheckPositiveInt("transfer_duration", ent.TransferDuration)...)
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
			ent.AgencyID.Valid = true
		} else {
			return causes.NewInvalidReferenceError("agency_id", ent.AgencyID.Key)
		}
	}
	return nil
}
