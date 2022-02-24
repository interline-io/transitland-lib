package plus

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
)

// FareRiderCategory fare_rider_categories.txt
type FareRiderCategory struct {
	FareID           string  `csv:"fare_id"`
	RiderCategoryID  int     `csv:"rider_category_id"`
	Price            float64 `csv:"price"`
	ExpirationDate   tl.Date `csv:"expiration_date"`
	CommencementDate tl.Date `csv:"commencement_date"`
	tl.BaseEntity
}

// Filename fare_rider_categories.txt
func (ent *FareRiderCategory) Filename() string {
	return "fare_rider_categories.txt"
}

// TableName ext_plus_fare_attributes
func (ent *FareRiderCategory) TableName() string {
	return "ext_plus_fare_rider_categories"
}

// UpdateKeys updates Entity references.
func (ent *FareRiderCategory) UpdateKeys(emap *tl.EntityMap) error {
	if fkid, ok := emap.GetEntity(&tl.FareAttribute{FareID: ent.FareID}); ok {
		ent.FareID = fkid
	} else {
		return causes.NewInvalidReferenceError("fare_id", ent.FareID)
	}
	return nil
}
