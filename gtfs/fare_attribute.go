package gtfs

import (
	"github.com/interline-io/transitland-lib/tt"
)

// FareAttribute fare_attributes.txt
type FareAttribute struct {
	FareID           tt.String   `csv:",required" standardized_sort:"1"`
	Price            tt.Float    `csv:",required" range:"0,"`
	CurrencyType     tt.Currency `csv:",required"`
	PaymentMethod    tt.Int      `csv:",required" enum:"0,1"`
	Transfers        tt.Int      `enum:"0,1,2"` // note! null is distinct from 0
	AgencyID         tt.Key      `target:"agency.txt"`
	TransferDuration tt.Int      `range:"0,"`
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

// Filename fare_attributes.txt
func (ent *FareAttribute) Filename() string {
	return "fare_attributes.txt"
}

// TableName gtfs_fare_attributes
func (ent *FareAttribute) TableName() string {
	return "gtfs_fare_attributes"
}
