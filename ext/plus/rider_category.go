package plus

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
)

// PlusRiderCategory rider_categories.txt
type PlusRiderCategory struct {
	AgencyID                 string `csv:"agency_id"`
	RiderCategoryID          int    `csv:"rider_category_id"`
	RiderCategoryDescription string `csv:"rider_category_description"`
	tl.BaseEntity
}

// Filename rider_categories.txt
func (ent *PlusRiderCategory) Filename() string {
	return "mtc_rider_categories.txt"
}

// TableName ext_plus_rider_categories
func (ent *PlusRiderCategory) TableName() string {
	return "ext_plus_rider_categories"
}

// UpdateKeys updates Entity references.
func (ent *PlusRiderCategory) UpdateKeys(emap *tl.EntityMap) error {
	if len(ent.AgencyID) > 0 {
		if fkey, ok := emap.GetEntity(&tl.Agency{AgencyID: ent.AgencyID}); ok {
			ent.AgencyID = fkey
		} else {
			return causes.NewInvalidReferenceError("agency_id", ent.AgencyID)
		}
	}
	return nil
}
