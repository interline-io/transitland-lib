package tl

import (
	"github.com/interline-io/transitland-lib/tl/tt"
)

// RiderCategory rider_categories.txt
type RiderCategory struct {
	RiderCategoryID   string
	RiderCategoryName string
	MinAge            Int
	MaxAge            Int
	EligibilityURL    String
	BaseEntity
}

func (ent *RiderCategory) Filename() string {
	return "rider_categories.txt"
}

func (ent *RiderCategory) TableName() string {
	return "gtfs_rider_categories"
}

func (ent *RiderCategory) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, tt.CheckPresent("rider_category_id", ent.RiderCategoryID)...)
	errs = append(errs, tt.CheckPresent("rider_category_name", ent.RiderCategoryName)...)
	errs = append(errs, tt.CheckPositiveInt("min_age", ent.MinAge.Val)...)
	errs = append(errs, tt.CheckPositiveInt("min_age", ent.MaxAge.Val)...)
	errs = append(errs, tt.CheckURL("eligibility_url", ent.EligibilityURL.Val)...)
	// todo: min_age < max_age
	return errs
}
