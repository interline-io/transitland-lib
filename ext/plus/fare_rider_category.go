package plus

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// FareRiderCategory fare_rider_categories.txt
type FareRiderCategory struct {
	FareID           string  `csv:"fare_id"`
	RiderCategoryID  int     `csv:"rider_category_id"`
	Price            float64 `csv:"price"`
	ExpirationDate   tt.Date `csv:"expiration_date"`
	CommencementDate tt.Date `csv:"commencement_date"`
	tt.BaseEntity
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
func (ent *FareRiderCategory) UpdateKeys(emap *tt.EntityMap) error {
	if fkid, ok := emap.GetEntity(&gtfs.FareAttribute{FareID: ent.FareID}); ok {
		ent.FareID = fkid
	} else {
		return causes.NewInvalidReferenceError("fare_id", ent.FareID)
	}
	return nil
}
