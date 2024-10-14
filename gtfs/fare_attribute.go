package gtfs

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// FareAttribute fare_attributes.txt
type FareAttribute struct {
	FareID           tt.String
	Price            tt.Float
	CurrencyType     tt.String
	PaymentMethod    tt.Int
	Transfers        tt.Int
	AgencyID         tt.Key `target:"agency.txt"`
	TransferDuration tt.Int
	tt.BaseEntity
}

// EntityID returns the ID or FareID.
func (ent *FareAttribute) EntityID() string {
	return entID(ent.ID, ent.FareID.Val)
}

// EntityKey returns the GTFS identifier.
func (ent *FareAttribute) EntityKey() string {
	return ent.FareID.Val
}

// Errors for this Entity.
func (ent *FareAttribute) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, tt.CheckPresent("fare_id", ent.FareID.Val)...)
	errs = append(errs, tt.CheckPresent("currency_type", ent.CurrencyType.Val)...)

	if !ent.Price.Valid {
		errs = append(errs, causes.NewRequiredFieldError("price"))
	} else {
		errs = append(errs, tt.CheckPositive("price", ent.Price.Val)...)
	}

	if !ent.PaymentMethod.Valid {
		errs = append(errs, causes.NewRequiredFieldError("payment_method"))
	} else {
		errs = append(errs, tt.CheckInsideRangeInt("payment_method", ent.PaymentMethod.Val, 0, 1)...)
	}

	errs = append(errs, tt.CheckCurrency("currency_type", ent.CurrencyType.Val)...)
	errs = append(errs, tt.CheckPositiveInt("transfer_duration", ent.TransferDuration.Val)...)
	errs = append(errs, tt.CheckInsideRangeInt("transfers", int(ent.Transfers.Val), 0, 2)...)
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
