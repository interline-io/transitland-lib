package gql

import (
	"testing"
)

func TestOperatorResolver(t *testing.T) {
	testcases := []testcase{
		{
			name:   "basic fields",
			query:  `query{operators(where:{onestop_id:"o-9q9-bayarearapidtransit"}) {onestop_id}}`,
			vars:   hw{},
			expect: `{"operators":[{"onestop_id":"o-9q9-bayarearapidtransit"}]}`,
		},
		{
			name:         "feeds",
			query:        `query{operators(where:{onestop_id:"o-9q9-bayarearapidtransit"}) {feeds{onestop_id}}}`,
			selector:     "operators.0.feeds.#.onestop_id",
			selectExpect: []string{"BA"},
		},
		{
			name:         "feeds incl rt",
			query:        `query{operators(where:{onestop_id:"o-9q9-caltrain"}) {feeds{onestop_id}}}`,
			selector:     "operators.0.feeds.#.onestop_id",
			selectExpect: []string{"CT", "CT~rt"},
		},
		{
			name:         "feeds only gtfs-rt",
			query:        `query{operators(where:{onestop_id:"o-9q9-caltrain"}) {feeds(where:{spec:GTFS_RT}) {onestop_id}}}`,
			selector:     "operators.0.feeds.#.onestop_id",
			selectExpect: []string{"CT~rt"},
		},
		{
			name:         "feeds only gtfs",
			query:        `query{operators(where:{onestop_id:"o-9q9-caltrain"}) {feeds(where:{spec:GTFS}) {onestop_id}}}`,
			selector:     "operators.0.feeds.#.onestop_id",
			selectExpect: []string{"CT"},
		},
		{
			name:   "tags us_ntd_id=90134",
			query:  `query{operators(where:{tags:{us_ntd_id:"90134"}}) {onestop_id}}`,
			vars:   hw{},
			expect: `{"operators":[{"onestop_id":"o-9q9-caltrain"}]}`,
		},
		{
			name:   "tags us_ntd_id=12345",
			query:  `query{operators(where:{tags:{us_ntd_id:"12345"}}) {onestop_id}}`,
			vars:   hw{},
			expect: `{"operators":[]}`,
		},
		{
			name:   "tags us_ntd_id presence",
			query:  `query{operators(where:{tags:{us_ntd_id:""}}) {onestop_id}}`,
			vars:   hw{},
			expect: `{"operators":[{"onestop_id":"o-9q9-caltrain"}]}`,
		},
		{
			name:         "places iso3166 country",
			query:        `query { operators(where:{adm0_iso: "US"}) {onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit"},
		},
		{
			name:         "places iso3166 state",
			query:        `query { operators(where:{adm1_iso: "US-CA"}) {onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			name:         "places iso3166 state not found",
			query:        `query { operators(where:{adm1_iso: "US-NY"}) {onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{},
		},
		{
			name:         "places adm0_name",
			query:        `query { operators(where:{adm0_name: "United States of America"}) {onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit"},
		},
		{
			name:         "places adm1_name",
			query:        `query { operators(where:{adm1_name: "California"}) {onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			name:         "places adm1_name not found",
			query:        `query { operators(where:{adm1_name: "Nowhere"}) {onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{},
		},
		{
			name:         "places city_name",
			query:        `query { operators(where:{city_name: "Oakland"}) {onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
		// search
		{
			name:         "search by onestop id",
			query:        `query { operators(where:{search: "o-9q9-bayarearapidtransit"}) {onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			name:         "search by name",
			query:        `query { operators(where:{search: "bay area"}) {onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			name:         "search not found",
			query:        `query { operators(where:{search: "new york"}) {onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{},
		},
		// spatial
		{
			name:         "near 100m",
			query:        `query {operators(where:{near:{lon:-122.407974,lat:37.784471,radius:100.0}}) {onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			name:         "near 10000m",
			query:        `query {operators(where:{near:{lon:-122.407974,lat:37.784471,radius:10000.0}}) {onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{"o-9q9-caltrain", "o-9q9-bayarearapidtransit"},
		},
		{
			name:         "within polygon",
			query:        `query{operators(where:{within:{type:"Polygon",coordinates:[[[-122.39803791046143,37.794626736533836],[-122.40106344223022,37.792303711508595],[-122.3965573310852,37.789641468930114],[-122.3938751220703,37.792354581451946],[-122.39803791046143,37.794626736533836]]]}}){onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			name:         "within polygon big",
			query:        `query{operators(where:{within:{type:"Polygon",coordinates:[[[-122.39481925964355,37.80151060070086],[-122.41653442382812,37.78652126637423],[-122.39662170410156,37.76847577247014],[-122.37301826477051,37.784757615348575],[-122.39481925964355,37.80151060070086]]]}}){id onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{"o-9q9-caltrain", "o-9q9-bayarearapidtransit"},
		},
		{
			name:         "where bbox 1",
			query:        `query($bbox:BoundingBox) {operators(where:{bbox:$bbox}) {onestop_id}}`,
			vars:         hw{"bbox": hw{"min_lon": -122.2698781543005, "min_lat": 37.80700393130445, "max_lon": -122.2677640139239, "max_lat": 37.8088734037938}},
			selector:     "operators.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			name:         "where bbox 2",
			query:        `query($bbox:BoundingBox) {operators(where:{bbox:$bbox}) {onestop_id}}`,
			vars:         hw{"bbox": hw{"min_lon": -124.3340029563042, "min_lat": 40.65505368922123, "max_lon": -123.9653594784379, "max_lat": 40.896440342606525}},
			selector:     "operators.#.onestop_id",
			selectExpect: []string{},
		},
		{
			name:        "where bbox too large",
			query:       `query($bbox:BoundingBox) {operators(where:{bbox:$bbox}) {onestop_id}}`,
			vars:        hw{"bbox": hw{"min_lon": -137.88020156441956, "min_lat": 30.072648315782004, "max_lon": -109.00421121090919, "max_lat": 45.02437957865729}},
			expectError: true,
			f: func(t *testing.T, jj string) {
			},
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

func TestOperatorResolver_License(t *testing.T) {
	q := `
	query ($lic: LicenseFilter) {
		operators(limit: 10000, where: {license: $lic}) {
		  onestop_id
		}
	  }	  
	`
	selector := `operators.#.onestop_id`
	testcases := []testcase{
		// license: share_alike_optional
		{
			name:               "license filter: share_alike_optional = yes",
			query:              q,
			vars:               hw{"lic": hw{"share_alike_optional": "YES"}},
			selector:           selector,
			selectExpectUnique: []string{"o-dhv-hillsborougharearegionaltransit"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: share_alike_optional = no",
			query:              q,
			vars:               hw{"lic": hw{"share_alike_optional": "NO"}},
			selector:           selector,
			selectExpectUnique: []string{"o-9q9-bayarearapidtransit"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: share_alike_optional = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"share_alike_optional": "EXCLUDE_NO"}},
			selector:           selector,
			selectExpectUnique: []string{"o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-9qs-demotransitauthority", "o-c20-ctran-c~tran"},
			selectExpectCount:  4,
		},
		// license: create_derived_product
		{
			name:               "license filter: create_derived_product = yes",
			query:              q,
			vars:               hw{"lic": hw{"create_derived_product": "YES"}},
			selector:           selector,
			selectExpectUnique: []string{"o-dhv-hillsborougharearegionaltransit"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: create_derived_product = no",
			query:              q,
			vars:               hw{"lic": hw{"create_derived_product": "NO"}},
			selector:           selector,
			selectExpectUnique: []string{"o-9q9-bayarearapidtransit"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: create_derived_product = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"create_derived_product": "EXCLUDE_NO"}},
			selector:           selector,
			selectExpectUnique: []string{"o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-9qs-demotransitauthority", "o-c20-ctran-c~tran"},
			selectExpectCount:  4,
		},
		// license: commercial_use_allowed
		{
			name:               "license filter: commercial_use_allowed = yes",
			query:              q,
			vars:               hw{"lic": hw{"commercial_use_allowed": "YES"}},
			selector:           selector,
			selectExpectUnique: []string{"o-dhv-hillsborougharearegionaltransit"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: commercial_use_allowed = no",
			query:              q,
			vars:               hw{"lic": hw{"commercial_use_allowed": "NO"}},
			selector:           selector,
			selectExpectUnique: []string{"o-9q9-bayarearapidtransit"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: commercial_use_allowed = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"commercial_use_allowed": "EXCLUDE_NO"}},
			selector:           selector,
			selectExpectUnique: []string{"o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-9qs-demotransitauthority", "o-c20-ctran-c~tran"},
			selectExpectCount:  4,
		},
		// license: redistribution_allowed
		{
			name:               "license filter: redistribution_allowed = yes",
			query:              q,
			vars:               hw{"lic": hw{"redistribution_allowed": "YES"}},
			selector:           selector,
			selectExpectUnique: []string{"o-dhv-hillsborougharearegionaltransit"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: redistribution_allowed = no",
			query:              q,
			vars:               hw{"lic": hw{"redistribution_allowed": "NO"}},
			selector:           selector,
			selectExpectUnique: []string{"o-9q9-bayarearapidtransit"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: redistribution_allowed = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"redistribution_allowed": "EXCLUDE_NO"}},
			selector:           selector,
			selectExpectUnique: []string{"o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-9qs-demotransitauthority", "o-c20-ctran-c~tran"},
			selectExpectCount:  4,
		},
		// license: use_without_attribution
		{
			name:               "license filter: use_without_attribution = yes",
			query:              q,
			vars:               hw{"lic": hw{"use_without_attribution": "YES"}},
			selector:           selector,
			selectExpectUnique: []string{"o-dhv-hillsborougharearegionaltransit"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: use_without_attribution = no",
			query:              q,
			vars:               hw{"lic": hw{"use_without_attribution": "NO"}},
			selector:           selector,
			selectExpectUnique: []string{"o-9q9-bayarearapidtransit"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: use_without_attribution = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"use_without_attribution": "EXCLUDE_NO"}},
			selector:           selector,
			selectExpectUnique: []string{"o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-9qs-demotransitauthority", "o-c20-ctran-c~tran"},
			selectExpectCount:  4,
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}
