package gqltest

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
)

func queryResolverTestcases(t testing.TB, cfg model.Config) []testcase {
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
	return testcases
}
