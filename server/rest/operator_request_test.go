package rest

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
)

func TestOperatorRequest(t *testing.T) {
	testcases := []testCase{
		{
			name:         "basic",
			h:            OperatorRequest{},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-9q9-caltrain", "o-c20-ctran-c~tran", "o-9q9-bayarearapidtransit", "o-dhv-hillsborougharearegionaltransit", "o-9qs-demotransitauthority"},
		},
		{
			name:         "feed_onestop_id",
			h:            OperatorRequest{FeedOnestopID: "BA"},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			name:         "onestop_id",
			h:            OperatorRequest{OnestopID: "o-9q9-bayarearapidtransit"},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			name:         "search",
			h:            OperatorRequest{Search: "bay area"},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			name:         "tags us_ntd_id=90134",
			h:            OperatorRequest{TagKey: "us_ntd_id", TagValue: "90134"},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-9q9-caltrain"},
		},
		{
			name:         "tags us_ntd_id present",
			h:            OperatorRequest{TagKey: "us_ntd_id", TagValue: ""},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-9q9-caltrain"},
		},
		{
			name:         "adm0name",
			h:            OperatorRequest{Adm0Name: "united states of america"},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-9q9-caltrain", "o-9q9-bayarearapidtransit", "o-dhv-hillsborougharearegionaltransit"},
		},
		{
			name:         "adm1name",
			h:            OperatorRequest{Adm1Name: "california"},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-9q9-caltrain", "o-9q9-bayarearapidtransit"},
		},
		{
			name:         "adm0iso",
			h:            OperatorRequest{Adm0Iso: "us"},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-9q9-caltrain", "o-9q9-bayarearapidtransit", "o-dhv-hillsborougharearegionaltransit"},
		},
		{
			name:         "adm1iso:us-ca",
			h:            OperatorRequest{Adm1Iso: "us-ca"},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-9q9-caltrain", "o-9q9-bayarearapidtransit"},
		},
		{
			name:         "adm1iso:us-ny",
			h:            OperatorRequest{Adm1Iso: "us-ny"},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{},
		},
		{
			name:         "city_name:san jose",
			h:            OperatorRequest{CityName: "san jose"},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-9q9-caltrain"},
		},
		{
			name:         "city_name:oakland",
			h:            OperatorRequest{CityName: "berkeley"},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			name:         "city_name:new york city",
			h:            OperatorRequest{CityName: "new york city"},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{},
		},
		{
			name:         "lat,lon,radius 10m",
			h:            OperatorRequest{Lon: -122.407974, Lat: 37.784471, Radius: 10},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			name:         "lat,lon,radius 2000m",
			h:            OperatorRequest{Lon: -122.407974, Lat: 37.784471, Radius: 2000},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-9q9-caltrain", "o-9q9-bayarearapidtransit"},
		},
		{
			name:         "lat,lon,radius florida",
			h:            OperatorRequest{Lon: -82.45857, Lat: 27.94798, Radius: 1000},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-dhv-hillsborougharearegionaltransit"},
		},
		{
			name:         "lat,lon,radius new york empty",
			h:            OperatorRequest{Lon: -74.00681230709345, Lat: 40.71335722414244, Radius: 1000},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{},
		},
		{
			name:         "bbox",
			h:            OperatorRequest{Bbox: &restBbox{model.BoundingBox{MinLon: -122.2698781543005, MinLat: 37.80700393130445, MaxLon: -122.2677640139239, MaxLat: 37.8088734037938}}},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			name:         "bbox larger",
			h:            OperatorRequest{Bbox: &restBbox{model.BoundingBox{MinLon: -122.774406, MinLat: 37.541086, MaxLon: -121.895500, MaxLat: 37.966172}}},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-9q9-caltrain", "o-9q9-bayarearapidtransit"},
		},
		{
			name:        "bbox too large",
			h:           OperatorRequest{Bbox: &restBbox{model.BoundingBox{MinLon: -148.664396, MinLat: 4.999277, MaxLon: -36.164396, MaxLat: 59.173810}}},
			expectError: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			checkTestCase(t, tc)
		})
	}
}

func TestOperatorRequest_Pagination(t *testing.T) {
	testcases := []testCase{
		{
			name:         "limit:1",
			h:            OperatorRequest{WithCursor: WithCursor{Limit: 1}},
			selector:     "operators.#.onestop_id",
			expectLength: 1,
		},
		{
			name:         "limit:1000",
			h:            OperatorRequest{WithCursor: WithCursor{Limit: 1000}},
			selector:     "operators.#.onestop_id",
			expectLength: 5,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			checkTestCase(t, tc)
		})
	}
}

func TestOperatorRequest_License(t *testing.T) {
	testcases := []testCase{
		{
			name:         "license:share_alike_optional yes",
			h:            OperatorRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseShareAlikeOptional: "yes"}},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-dhv-hillsborougharearegionaltransit"},
		},
		{
			name:         "license:share_alike_optional no",
			h:            OperatorRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseShareAlikeOptional: "no"}},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			name:         "license:share_alike_optional exclude_no",
			h:            OperatorRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseShareAlikeOptional: "exclude_no"}},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-9q9-caltrain", "o-c20-ctran-c~tran", "o-dhv-hillsborougharearegionaltransit", "o-9qs-demotransitauthority"},
		},
		{
			name:         "license:commercial_use_allowed yes",
			h:            OperatorRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseCommercialUseAllowed: "yes"}},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-dhv-hillsborougharearegionaltransit"},
		},
		{
			name:         "license:commercial_use_allowed no",
			h:            OperatorRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseCommercialUseAllowed: "no"}},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			name:         "license:commercial_use_allowed exclude_no",
			h:            OperatorRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseCommercialUseAllowed: "exclude_no"}},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-9q9-caltrain", "o-c20-ctran-c~tran", "o-dhv-hillsborougharearegionaltransit", "o-9qs-demotransitauthority"},
		},
		{
			name:         "license:create_derived_product yes",
			h:            OperatorRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseCreateDerivedProduct: "yes"}},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-dhv-hillsborougharearegionaltransit"},
		},
		{
			name:         "license:create_derived_product no",
			h:            OperatorRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseCreateDerivedProduct: "no"}},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			name:         "license:create_derived_product exclude_no",
			h:            OperatorRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseCreateDerivedProduct: "exclude_no"}},
			selector:     "operators.#.onestop_id",
			expectSelect: []string{"o-9q9-caltrain", "o-c20-ctran-c~tran", "o-dhv-hillsborougharearegionaltransit", "o-9qs-demotransitauthority"},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			checkTestCase(t, tc)
		})
	}
}
