package findertest

// Cases duplicated from server/gql/operator_resolver_test.go.

func operatorResolverLicenseTestcases() []testcase {
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
			selectExpectUnique: []string{"o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata"},
			selectExpectCount:  2,
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
			selectExpectUnique: []string{"o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata", "o-9qs-demotransitauthority", "o-unknown-c~tran"},
			selectExpectCount:  5,
		},
		// license: create_derived_product
		{
			name:               "license filter: create_derived_product = yes",
			query:              q,
			vars:               hw{"lic": hw{"create_derived_product": "YES"}},
			selector:           selector,
			selectExpectUnique: []string{"o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata"},
			selectExpectCount:  2,
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
			selectExpectUnique: []string{"o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata", "o-9qs-demotransitauthority", "o-unknown-c~tran"},
			selectExpectCount:  5,
		},
		// license: commercial_use_allowed
		{
			name:               "license filter: commercial_use_allowed = yes",
			query:              q,
			vars:               hw{"lic": hw{"commercial_use_allowed": "YES"}},
			selector:           selector,
			selectExpectUnique: []string{"o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata"},
			selectExpectCount:  2,
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
			selectExpectUnique: []string{"o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata", "o-9qs-demotransitauthority", "o-unknown-c~tran"},
			selectExpectCount:  5,
		},
		// license: redistribution_allowed
		{
			name:               "license filter: redistribution_allowed = yes",
			query:              q,
			vars:               hw{"lic": hw{"redistribution_allowed": "YES"}},
			selector:           selector,
			selectExpectUnique: []string{"o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata"},
			selectExpectCount:  2,
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
			selectExpectUnique: []string{"o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata", "o-9qs-demotransitauthority", "o-unknown-c~tran"},
			selectExpectCount:  5,
		},
		// license: use_without_attribution
		{
			name:               "license filter: use_without_attribution = yes",
			query:              q,
			vars:               hw{"lic": hw{"use_without_attribution": "YES"}},
			selector:           selector,
			selectExpectUnique: []string{"o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata"},
			selectExpectCount:  2,
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
			selectExpectUnique: []string{"o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata", "o-9qs-demotransitauthority", "o-unknown-c~tran"},
			selectExpectCount:  5,
		},
	}
	return testcases
}
