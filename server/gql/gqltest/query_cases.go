package gqltest

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
)

func queryResolverTestcases(t testing.TB, cfg model.Config) []testcase {
	q := `query{me{id name email external_data roles}}`
	testcases := []testcase{
		{
			Name:         "basic",
			Query:        q,
			Selector:     "me.id",
			SelectExpect: []string{"testuser"},
		},
		{
			Name:         "basic",
			Query:        q,
			Selector:     "me.roles",
			SelectExpect: []string{"testrole"},
		},
	}
	return testcases
}
