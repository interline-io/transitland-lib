package gtfs

import (
	"fmt"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// RiderCategory rider_categories.txt
type RiderCategory struct {
	RiderCategoryID   tt.String `csv:",required"`
	RiderCategoryName tt.String `csv:",required"`
	MinAge            tt.Int    `range:"0,"`
	MaxAge            tt.Int    `range:"0,"`
	EligibilityURL    tt.Url
	tt.BaseEntity
}

func (ent *RiderCategory) Filename() string {
	return "rider_categories.txt"
}

func (ent *RiderCategory) TableName() string {
	return "gtfs_rider_categories"
}

func (ent *RiderCategory) GroupKey() (string, string) {
	return "rider_category_id", ent.RiderCategoryID.Val
}

func (ent *RiderCategory) ConditionalErrors() (errs []error) {
	if ent.MinAge.Valid && ent.MaxAge.Valid && ent.MaxAge.Val < ent.MinAge.Val {
		errs = append(errs, causes.NewInvalidFieldError("max_age", ent.MaxAge.String(), fmt.Errorf("max_age is less than min_age")))
	}
	return errs
}
