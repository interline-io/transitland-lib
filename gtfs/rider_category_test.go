package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tt"
)

func TestRiderCategory_Errors(t *testing.T) {
	newRiderCategory := func(fn func(*RiderCategory)) *RiderCategory {
		riderCategory := &RiderCategory{
			RiderCategoryID:       tt.NewString("ok"),
			RiderCategoryName:     tt.NewString("ok"),
			MinAge:                tt.Int{},
			MaxAge:                tt.Int{},
			IsDefaultFareCategory: tt.Int{},
			EligibilityURL:        tt.Url{},
		}
		if fn != nil {
			fn(riderCategory)
		}
		return riderCategory
	}

	testcases := []struct {
		name           string
		entity         *RiderCategory
		expectedErrors []testutil.ExpectError
	}{
		{
			name:           "Valid rider_category",
			entity:         newRiderCategory(nil),
			expectedErrors: nil,
		},
		{
			name: "Missing rider_category_id",
			entity: newRiderCategory(func(rc *RiderCategory) {
				rc.RiderCategoryID = tt.String{}
			}),
			expectedErrors: ParseExpectErrors("RequiredFieldError:rider_category_id"),
		},
		{
			name: "Missing rider_category_name",
			entity: newRiderCategory(func(rc *RiderCategory) {
				rc.RiderCategoryName = tt.String{}
			}),
			expectedErrors: ParseExpectErrors("RequiredFieldError:rider_category_name"),
		},
		{
			name: "Invalid min_age (negative)",
			entity: newRiderCategory(func(rc *RiderCategory) {
				rc.MinAge = tt.NewInt(-1)
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:min_age"),
		},
		{
			name: "Invalid max_age (negative)",
			entity: newRiderCategory(func(rc *RiderCategory) {
				rc.MaxAge = tt.NewInt(-1)
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:max_age"),
		},
		{
			name: "max_age less than min_age",
			entity: newRiderCategory(func(rc *RiderCategory) {
				rc.MinAge = tt.NewInt(10)
				rc.MaxAge = tt.NewInt(5)
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:max_age"),
		},
		{
			name: "Invalid eligibility_url",
			entity: newRiderCategory(func(rc *RiderCategory) {
				rc.EligibilityURL = tt.NewUrl("asd")
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:eligibility_url"),
		},
		{
			name: "Invalid is_default_fare_category",
			entity: newRiderCategory(func(rc *RiderCategory) {
				rc.IsDefaultFareCategory = tt.NewInt(2)
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:is_default_fare_category"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.entity)
			testutil.CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}
