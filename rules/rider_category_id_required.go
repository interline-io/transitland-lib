package rules

import (
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

type fareProductRiderCategories struct {
	riderCategories mapset.Set[string]
}

// FareProductRiderCategoryDefaultCheck checks if agency_id is missing when more than one agency is present.
type FareProductRiderCategoryDefaultCheck struct {
	riderCategoryIsDefault     map[string]int64
	fareProductRiderCategories map[string]fareProductRiderCategories
}

// Validate checks that no fare_product_id group allows more than one default rider category.
func (e *FareProductRiderCategoryDefaultCheck) Validate(ent tt.Entity) []error {
	// Lazy map initialization
	if e.riderCategoryIsDefault == nil {
		e.riderCategoryIsDefault = make(map[string]int64)
	}
	if e.fareProductRiderCategories == nil {
		e.fareProductRiderCategories = make(map[string]fareProductRiderCategories)
	}
	//
	var errs []error
	switch v := ent.(type) {
	case *gtfs.RiderCategory:
		e.riderCategoryIsDefault[v.RiderCategoryID.Val] = v.IsDefaultFareCategory.Val
	case *gtfs.FareProduct:
		fid := v.FareProductID.Val
		a, ok := e.fareProductRiderCategories[fid]
		if !ok {
			a = fareProductRiderCategories{riderCategories: mapset.NewSet[string]()}
		}

		// Add the rider category to the set of rider categories for this fare product
		if v.RiderCategoryID.Valid {
			a.riderCategories.Add(v.RiderCategoryID.Val)
		} else {
			// If the rider category is not valid, we need to add all default rider categories
			for k := range e.riderCategoryIsDefault {
				a.riderCategories.Add(k)
			}
		}

		// Check if there are multiple default rider categories
		var conflictingCategories []string
		for _, k := range a.riderCategories.ToSlice() {
			if e.riderCategoryIsDefault[k] == 1 {
				conflictingCategories = append(conflictingCategories, k)
			}
		}
		if len(conflictingCategories) > 1 {
			errs = append(errs, causes.NewAmbiguousRiderCategoryError(fid, conflictingCategories...))
		}
		e.fareProductRiderCategories[fid] = a
	}
	return errs
}
