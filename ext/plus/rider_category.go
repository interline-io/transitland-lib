package plus

import (
	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/causes"
)

// RiderCategory rider_categories.txt
type RiderCategory struct {
	AgencyID                 string `csv:"agency_id"`
	RiderCategoryID          int    `csv:"rider_category_id"`
	RiderCategoryDescription string `csv:"rider_category_description"`
	ucount                   int
	gotransit.BaseEntity
}

// Filename rider_categories.txt
func (ent *RiderCategory) Filename() string {
	return "rider_categories.txt"
}

// TableName ext_plus_rider_categories
func (ent *RiderCategory) TableName() string {
	return "ext_plus_rider_categories"
}

// UpdateKeys updates Entity references.
func (ent *RiderCategory) UpdateKeys(emap *gotransit.EntityMap) error {
	if len(ent.AgencyID) > 0 {
		if fkey, ok := emap.Get(&gotransit.Agency{AgencyID: ent.AgencyID}); ok {
			ent.AgencyID = fkey
		} else {
			return causes.NewInvalidReferenceError("agency_id", ent.AgencyID)
		}
	}
	return nil
}
