package gqltest

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
)

func operatorResolverLicenseTestcases(t testing.TB, cfg model.Config) []testcase {
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
			Name:               "license filter: share_alike_optional = yes",
			Query:              q,
			Vars:               hw{"lic": hw{"share_alike_optional": "YES"}},
			Selector:           selector,
			SelectExpectUnique: []string{"o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata"},
			SelectExpectCount:  2,
		},
		{
			Name:               "license filter: share_alike_optional = no",
			Query:              q,
			Vars:               hw{"lic": hw{"share_alike_optional": "NO"}},
			Selector:           selector,
			SelectExpectUnique: []string{"o-9q9-bayarearapidtransit"},
			SelectExpectCount:  1,
		},
		{
			Name:               "license filter: share_alike_optional = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"share_alike_optional": "EXCLUDE_NO"}},
			Selector:           selector,
			SelectExpectUnique: []string{"o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata", "o-9qs-demotransitauthority", "o-unknown-c~tran"},
			SelectExpectCount:  5,
		},
		// license: create_derived_product
		{
			Name:               "license filter: create_derived_product = yes",
			Query:              q,
			Vars:               hw{"lic": hw{"create_derived_product": "YES"}},
			Selector:           selector,
			SelectExpectUnique: []string{"o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata"},
			SelectExpectCount:  2,
		},
		{
			Name:               "license filter: create_derived_product = no",
			Query:              q,
			Vars:               hw{"lic": hw{"create_derived_product": "NO"}},
			Selector:           selector,
			SelectExpectUnique: []string{"o-9q9-bayarearapidtransit"},
			SelectExpectCount:  1,
		},
		{
			Name:               "license filter: create_derived_product = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"create_derived_product": "EXCLUDE_NO"}},
			Selector:           selector,
			SelectExpectUnique: []string{"o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata", "o-9qs-demotransitauthority", "o-unknown-c~tran"},
			SelectExpectCount:  5,
		},
		// license: commercial_use_allowed
		{
			Name:               "license filter: commercial_use_allowed = yes",
			Query:              q,
			Vars:               hw{"lic": hw{"commercial_use_allowed": "YES"}},
			Selector:           selector,
			SelectExpectUnique: []string{"o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata"},
			SelectExpectCount:  2,
		},
		{
			Name:               "license filter: commercial_use_allowed = no",
			Query:              q,
			Vars:               hw{"lic": hw{"commercial_use_allowed": "NO"}},
			Selector:           selector,
			SelectExpectUnique: []string{"o-9q9-bayarearapidtransit"},
			SelectExpectCount:  1,
		},
		{
			Name:               "license filter: commercial_use_allowed = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"commercial_use_allowed": "EXCLUDE_NO"}},
			Selector:           selector,
			SelectExpectUnique: []string{"o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata", "o-9qs-demotransitauthority", "o-unknown-c~tran"},
			SelectExpectCount:  5,
		},
		// license: redistribution_allowed
		{
			Name:               "license filter: redistribution_allowed = yes",
			Query:              q,
			Vars:               hw{"lic": hw{"redistribution_allowed": "YES"}},
			Selector:           selector,
			SelectExpectUnique: []string{"o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata"},
			SelectExpectCount:  2,
		},
		{
			Name:               "license filter: redistribution_allowed = no",
			Query:              q,
			Vars:               hw{"lic": hw{"redistribution_allowed": "NO"}},
			Selector:           selector,
			SelectExpectUnique: []string{"o-9q9-bayarearapidtransit"},
			SelectExpectCount:  1,
		},
		{
			Name:               "license filter: redistribution_allowed = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"redistribution_allowed": "EXCLUDE_NO"}},
			Selector:           selector,
			SelectExpectUnique: []string{"o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata", "o-9qs-demotransitauthority", "o-unknown-c~tran"},
			SelectExpectCount:  5,
		},
		// license: use_without_attribution
		{
			Name:               "license filter: use_without_attribution = yes",
			Query:              q,
			Vars:               hw{"lic": hw{"use_without_attribution": "YES"}},
			Selector:           selector,
			SelectExpectUnique: []string{"o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata"},
			SelectExpectCount:  2,
		},
		{
			Name:               "license filter: use_without_attribution = no",
			Query:              q,
			Vars:               hw{"lic": hw{"use_without_attribution": "NO"}},
			Selector:           selector,
			SelectExpectUnique: []string{"o-9q9-bayarearapidtransit"},
			SelectExpectCount:  1,
		},
		{
			Name:               "license filter: use_without_attribution = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"use_without_attribution": "EXCLUDE_NO"}},
			Selector:           selector,
			SelectExpectUnique: []string{"o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata", "o-9qs-demotransitauthority", "o-unknown-c~tran"},
			SelectExpectCount:  5,
		},
	}
	return testcases
}

func operatorResolverTestcases(t testing.TB, cfg model.Config) []testcase {
	testcases := []testcase{
		{
			Name:   "basic fields",
			Query:  `query{operators(where:{onestop_id:"o-9q9-bayarearapidtransit"}) {onestop_id}}`,
			Vars:   hw{},
			Expect: `{"operators":[{"onestop_id":"o-9q9-bayarearapidtransit"}]}`,
		},
		{
			Name:         "onestop_ids",
			Query:        `query{operators(where:{onestop_ids:["o-9q9-bayarearapidtransit","o-9q9-caltrain"]}) {onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			Name:         "onestop_ids with onestop_id",
			Query:        `query{operators(where:{onestop_id:"o-9q9-caltrain", onestop_ids:["o-9q9-bayarearapidtransit"]}) {onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			// BART is a virtual (generated) operator record in the test data; Caltrain is defined in Atlas.
			Name:  "generated",
			Query: `query{operators(where:{onestop_ids:["o-9q9-bayarearapidtransit","o-9q9-caltrain"]}) {onestop_id generated}}`,
			Sel: []testcaseSelector{
				{Selector: `operators.#(onestop_id=="o-9q9-bayarearapidtransit").generated`, Expect: []string{"true"}},
				{Selector: `operators.#(onestop_id=="o-9q9-caltrain").generated`, Expect: []string{"false"}},
			},
		},
		{
			Name:         "onestop_ids no match",
			Query:        `query{operators(where:{onestop_ids:["o-does-not-exist"]}) {onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{},
		},
		{
			Name:         "agencies",
			Query:        `query{operators(where:{onestop_id:"o-9q9-bayarearapidtransit"}) {agencies{agency_name}}}`,
			Selector:     "operators.0.agencies.#.agency_name",
			SelectExpect: []string{"Bay Area Rapid Transit"},
		},
		{
			Name:         "agencies limit",
			Query:        `query{operators(where:{onestop_id:"o-9q9-bayarearapidtransit"}) {agencies(limit:1){agency_name}}}`,
			Selector:     "operators.0.agencies.#.agency_name",
			SelectExpect: []string{"Bay Area Rapid Transit"},
		},
		{
			// Each operator gets its own agency: the limit counts per operator, and a
			// limit on the batch query would leave whichever operator came second empty.
			Name:  "agencies limit across operators",
			Query: `query{operators(where:{onestop_ids:["o-9q9-bayarearapidtransit","o-9q9-caltrain"]}) {onestop_id agencies(limit:1){agency_name}}}`,
			Sel: []testcaseSelector{
				{Selector: `operators.#(onestop_id=="o-9q9-bayarearapidtransit").agencies.#.agency_name`, Expect: []string{"Bay Area Rapid Transit"}},
				{Selector: `operators.#(onestop_id=="o-9q9-caltrain").agencies.#.agency_name`, Expect: []string{"Caltrain"}},
			},
		},
		{
			Name:         "agencies limit 0",
			Query:        `query{operators(where:{onestop_id:"o-9q9-bayarearapidtransit"}) {agencies(limit:0){agency_name}}}`,
			Selector:     "operators.0.agencies.#.agency_name",
			SelectExpect: []string{},
		},
		{
			Name:         "feeds",
			Query:        `query{operators(where:{onestop_id:"o-9q9-bayarearapidtransit"}) {feeds{onestop_id}}}`,
			Selector:     "operators.0.feeds.#.onestop_id",
			SelectExpect: []string{"BA"},
		},
		{
			Name:         "feeds incl rt",
			Query:        `query{operators(where:{onestop_id:"o-9q9-caltrain"}) {feeds{onestop_id}}}`,
			Selector:     "operators.0.feeds.#.onestop_id",
			SelectExpect: []string{"CT", "CT~rt"},
		},
		{
			Name:         "feeds only gtfs-rt",
			Query:        `query{operators(where:{onestop_id:"o-9q9-caltrain"}) {feeds(where:{spec:GTFS_RT}) {onestop_id}}}`,
			Selector:     "operators.0.feeds.#.onestop_id",
			SelectExpect: []string{"CT~rt"},
		},
		{
			Name:         "feeds only gtfs",
			Query:        `query{operators(where:{onestop_id:"o-9q9-caltrain"}) {feeds(where:{spec:GTFS}) {onestop_id}}}`,
			Selector:     "operators.0.feeds.#.onestop_id",
			SelectExpect: []string{"CT"},
		},
		{
			Name:   "tags us_ntd_id=90134",
			Query:  `query{operators(where:{tags:{us_ntd_id:"90134"}}) {onestop_id}}`,
			Vars:   hw{},
			Expect: `{"operators":[{"onestop_id":"o-9q9-caltrain"}]}`,
		},
		{
			Name:   "tags us_ntd_id=12345",
			Query:  `query{operators(where:{tags:{us_ntd_id:"12345"}}) {onestop_id}}`,
			Vars:   hw{},
			Expect: `{"operators":[]}`,
		},
		{
			Name:   "tags us_ntd_id presence",
			Query:  `query{operators(where:{tags:{us_ntd_id:""}}) {onestop_id}}`,
			Vars:   hw{},
			Expect: `{"operators":[{"onestop_id":"o-9q9-caltrain"}]}`,
		},
		// search
		{
			Name:         "search by onestop id",
			Query:        `query { operators(where:{search: "o-9q9-bayarearapidtransit"}) {onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			Name:         "search by name",
			Query:        `query { operators(where:{search: "bay area"}) {onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			Name:         "search not found",
			Query:        `query { operators(where:{search: "new york"}) {onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{},
		},
		{
			// `_` and `%` are ILIKE wildcards; unescaped they over-match every row.
			Name:         "search escapes ilike wildcards",
			Query:        `query { operators(where:{search: "__"}) {onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{},
		},
	}
	return testcases
}
