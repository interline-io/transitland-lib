package tl

import (
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/enum"
)

// FareAttribute fare_attributes.txt
type FareAttribute struct {
	FareID           string
	Price            Float
	CurrencyType     Currency
	PaymentMethod    IntEnum
	Transfers        IntEnum
	TransferDuration Int
	AgencyID         Key
	BaseEntity
}

// EntityID returns the ID or FareID.
func (ent *FareAttribute) EntityID() string {
	return entID(ent.ID, ent.FareID)
}

// EntityKey returns the GTFS identifier.
func (ent *FareAttribute) EntityKey() string {
	return ent.FareID
}

// Errors for this Entity.
func (ent *FareAttribute) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, enum.CheckPresent("fare_id", ent.FareID)...)
	errs = CheckError(errs, CheckValidPresent("currency_type", &ent.CurrencyType))
	errs = CheckError(errs, CheckValidPresent("price", &ent.Price))
	errs = CheckError(errs, CheckValidPresent("payment_method", &ent.PaymentMethod))
	errs = append(errs, enum.CheckPositive("price", ent.Price.Float)...)
	errs = append(errs, enum.CheckInsideRangeInt("payment_method", ent.PaymentMethod.Int, 0, 1)...)
	errs = append(errs, enum.CheckPositiveInt("transfer_duration", ent.TransferDuration.Int)...)
	errs = append(errs, enum.CheckInsideRangeInt("transfers", int(ent.Transfers.Int), 0, 2)...)
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
