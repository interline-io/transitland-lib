package gqltest

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/model"
)

func feedVersionResolverTranslationsTestcases(t testing.TB, cfg model.Config) []testcase {
	supplementalSha1 := testconfig.SupplementalSha1(t, cfg)
	// The supplemental feed has two translations by record_id for one stop, one by
	// field_value, and one for a route.
	q := `query($sha1:String!, $where:TranslationFilter) {
		feed_versions(where:{sha1:$sha1}) {
			translations(where:$where) { table_name field_name language translation record_id record_sub_id field_value }
		}
	}`
	testcases := []testcase{
		{
			Name:  "every translation in the feed version",
			Query: `query($sha1:String!) { feed_versions(where:{sha1:$sha1}) { translations { language feed_version_sha1 feed_onestop_id feed_version { sha1 } } } }`,
			Vars:  hw{"sha1": supplementalSha1},
			Sel: []testcaseSelector{
				{Selector: "feed_versions.0.translations.#.language", Expect: []string{"es", "fr", "es", "es"}},
				{Selector: "feed_versions.0.translations.#.feed_version_sha1", ExpectUnique: []string{supplementalSha1}},
				{Selector: "feed_versions.0.translations.#.feed_onestop_id", ExpectUnique: []string{"supplemental"}},
				{Selector: "feed_versions.0.translations.#.feed_version.sha1", ExpectUnique: []string{supplementalSha1}},
			},
		},
		{
			Name:   "table_name",
			Query:  q,
			Vars:   hw{"sha1": supplementalSha1, "where": hw{"table_name": "routes"}},
			Expect: `{"feed_versions":[{"translations":[{"table_name":"routes","field_name":"route_long_name","language":"es","translation":"Servicio local","record_id":"L","record_sub_id":null,"field_value":null}]}]}`,
		},
		{
			Name:         "record_id",
			Query:        q,
			Vars:         hw{"sha1": supplementalSha1, "where": hw{"record_id": "N1"}},
			Selector:     "feed_versions.0.translations.#.translation",
			SelectExpect: []string{"Estación Norte, andén 1", "Gare Nord, quai 1"},
		},
		{
			Name:  "language and field_name",
			Query: q,
			Vars:  hw{"sha1": supplementalSha1, "where": hw{"language": "es", "field_name": "stop_name"}},
			Sel: []testcaseSelector{
				{Selector: "feed_versions.0.translations.#.translation", Expect: []string{"Estación Norte, andén 1", "Estación Sur, andén 1"}},
				{Selector: `feed_versions.0.translations.#(field_value=="South Station Platform 1")#.translation`, Expect: []string{"Estación Sur, andén 1"}},
			},
		},
		{
			// BCP 47 language tags are case-insensitive.
			Name:         "language in another case",
			Query:        q,
			Vars:         hw{"sha1": supplementalSha1, "where": hw{"language": "ES", "field_name": "stop_name"}},
			Selector:     "feed_versions.0.translations.#.translation",
			SelectExpect: []string{"Estación Norte, andén 1", "Estación Sur, andén 1"},
		},
		{
			Name:   "feed version without translations",
			Query:  q,
			Vars:   hw{"sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0"},
			Expect: `{"feed_versions":[{"translations":[]}]}`,
		},
	}
	return testcases
}
