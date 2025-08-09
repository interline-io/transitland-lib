package gql

import "testing"

func TestQueryResolver(t *testing.T) {
	q := `query{me{id name email external_data roles}}`
	testcases := []testcase{
		{
			name:         "basic",
			query:        q,
			selector:     "me.id",
			selectExpect: []string{"testuser"},
		},
		{
			name:         "basic",
			query:        q,
			selector:     "me.roles",
			selectExpect: []string{"testrole"},
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}
