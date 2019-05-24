package plus

import (
	"time"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/causes"
)

// FareRiderCategory fare_rider_categories.txt
type FareRiderCategory struct {
	FareID           string    `csv:"fare_id" gorm:"type:int"`
	RiderCategoryID  int       `csv:"rider_category_id"`
	Price            float64   `csv:"price"`
	ExpirationDate   time.Time `csv:"expiration_date"`
	CommencementDate time.Time `csv:"commencement_date"`
	gotransit.BaseEntity
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
func (ent *FareRiderCategory) UpdateKeys(emap *gotransit.EntityMap) error {
	if fkid, ok := emap.Get(&gotransit.FareAttribute{FareID: ent.FareID}); ok {
		ent.FareID = fkid
	} else {
		return causes.NewInvalidReferenceError("fare_id", ent.FareID)
	}
	return nil
}
