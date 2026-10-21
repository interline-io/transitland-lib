package findertest

// Cases duplicated from server/gql/agency_resolver_test.go.

func agencyResolverLicenseTestcases() []testcase {
	q := `
	query ($lic: LicenseFilter) {
		agencies(limit: 10000, where: {license: $lic}) {
		  agency_id
		  feed_version {
			feed {
			  onestop_id
			  license {
				share_alike_optional
				create_derived_product
				commercial_use_allowed
				redistribution_allowed
			  }
			}
		  }
		}
	  }	  
	`
	testcases := []testcase{
		// license: share_alike_optional
		{
			name:               "license filter: share_alike_optional = yes",
			query:              q,
			vars:               hw{"lic": hw{"share_alike_optional": "YES"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA", "WMATA"},
			selectExpectCount:  2,
		},
		{
			name:               "license filter: share_alike_optional = no",
			query:              q,
			vars:               hw{"lic": hw{"share_alike_optional": "NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: share_alike_optional = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"share_alike_optional": "EXCLUDE_NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			selectExpectCount:  4,
		},
		// license: create_derived_product
		{
			name:               "license filter: create_derived_product = yes",
			query:              q,
			vars:               hw{"lic": hw{"create_derived_product": "YES"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA", "WMATA"},
			selectExpectCount:  2,
		},
		{
			name:               "license filter: create_derived_product = no",
			query:              q,
			vars:               hw{"lic": hw{"create_derived_product": "NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: create_derived_product = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"create_derived_product": "EXCLUDE_NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			selectExpectCount:  4,
		},
		// license: commercial_use_allowed
		{
			name:               "license filter: commercial_use_allowed = yes",
			query:              q,
			vars:               hw{"lic": hw{"commercial_use_allowed": "YES"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA", "WMATA"},
			selectExpectCount:  2,
		},
		{
			name:               "license filter: commercial_use_allowed = no",
			query:              q,
			vars:               hw{"lic": hw{"commercial_use_allowed": "NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: commercial_use_allowed = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"commercial_use_allowed": "EXCLUDE_NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			selectExpectCount:  4,
		},
		// license: redistribution_allowed
		{
			name:               "license filter: redistribution_allowed = yes",
			query:              q,
			vars:               hw{"lic": hw{"redistribution_allowed": "YES"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA", "WMATA"},
			selectExpectCount:  2,
		},
		{
			name:               "license filter: redistribution_allowed = no",
			query:              q,
			vars:               hw{"lic": hw{"redistribution_allowed": "NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: redistribution_allowed = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"redistribution_allowed": "EXCLUDE_NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			selectExpectCount:  4,
		},
		// license: use_without_attribution
		{
			name:               "license filter: use_without_attribution = yes",
			query:              q,
			vars:               hw{"lic": hw{"use_without_attribution": "YES"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA", "WMATA"},
			selectExpectCount:  2,
		},
		{
			name:               "license filter: use_without_attribution = no",
			query:              q,
			vars:               hw{"lic": hw{"use_without_attribution": "NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: use_without_attribution = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"use_without_attribution": "EXCLUDE_NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			selectExpectCount:  4,
		},
	}
	return testcases
}
