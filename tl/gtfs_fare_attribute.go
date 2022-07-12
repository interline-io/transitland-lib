package tl

import (
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tt"
)

// FareAttribute fare_attributes.txt
type FareAttribute struct {
	FareID           string  `csv:",required"`
	Price            float64 `csv:",required"`
	CurrencyType     string  `csv:",required"`
	PaymentMethod    int     `csv:",required"`
	Transfers        Int
	AgencyID         Key
	TransferDuration int
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
	errs = append(errs, tt.CheckPresent("fare_id", ent.FareID)...)
	errs = append(errs, tt.CheckPresent("currency_type", ent.CurrencyType)...)
	errs = append(errs, tt.CheckPositive("price", ent.Price)...)
	errs = append(errs, tt.CheckCurrency("currency_type", ent.CurrencyType)...)
	errs = append(errs, tt.CheckInsideRangeInt("payment_method", ent.PaymentMethod, 0, 1)...)
	errs = append(errs, tt.CheckPositiveInt("transfer_duration", ent.TransferDuration)...)
	errs = append(errs, tt.CheckInsideRangeInt("transfers", int(ent.Transfers.Int), 0, 2)...)
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
