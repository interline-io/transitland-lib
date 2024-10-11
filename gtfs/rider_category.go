package gtfs

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tt"
)

// RiderCategory rider_categories.txt
type RiderCategory struct {
	RiderCategoryID   string
	RiderCategoryName string
	MinAge            tt.Int
	MaxAge            tt.Int
	EligibilityURL    tt.String
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
	errs = append(errs, tt.CheckPositiveInt("max_age", ent.MaxAge.Val)...)
	errs = append(errs, tt.CheckURL("eligibility_url", ent.EligibilityURL.Val)...)
	if ent.MinAge.Valid && ent.MaxAge.Valid && ent.MaxAge.Val < ent.MinAge.Val {
		errs = append(errs, causes.NewInvalidFieldError("max_age", tt.TryCsv(ent.MaxAge), fmt.Errorf("max_age is less than min_age")))
	}
	// todo: min_age < max_age
	return errs
}
