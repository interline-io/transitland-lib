package gql

import (
	"testing"
)

func TestFeedVersionResolver_Translations(t *testing.T) {
	// The test supplement adds CT translations: two by record_id for one stop,
	// one by field_value, and one for a route.
	ctSha1 := "d2813c293bcfd7a97dde599527ae6c62c98e66c6"
	q := `query($sha1:String!, $where:TranslationFilter) {
		feed_versions(where:{sha1:$sha1}) {
			translations(where:$where) { table_name field_name language translation record_id record_sub_id field_value }
		}
	}`
	testcases := []testcase{
		{
			name:  "every translation in the feed version",
			query: `query($sha1:String!) { feed_versions(where:{sha1:$sha1}) { translations { language feed_version_sha1 feed_onestop_id feed_version { sha1 } } } }`,
			vars:  hw{"sha1": ctSha1},
			sel: []testcaseSelector{
				{selector: "feed_versions.0.translations.#.language", expect: []string{"es", "fr", "es", "es"}},
				{selector: "feed_versions.0.translations.#.feed_version_sha1", expectUnique: []string{ctSha1}},
				{selector: "feed_versions.0.translations.#.feed_onestop_id", expectUnique: []string{"CT"}},
				{selector: "feed_versions.0.translations.#.feed_version.sha1", expectUnique: []string{ctSha1}},
			},
		},
		{
			name:   "table_name",
			query:  q,
			vars:   hw{"sha1": ctSha1, "where": hw{"table_name": "routes"}},
			expect: `{"feed_versions":[{"translations":[{"table_name":"routes","field_name":"route_long_name","language":"es","translation":"Servicio local","record_id":"Lo-130","record_sub_id":null,"field_value":null}]}]}`,
		},
		{
			name:         "record_id",
			query:        q,
			vars:         hw{"sha1": ctSha1, "where": hw{"record_id": "70011"}},
			selector:     "feed_versions.0.translations.#.translation",
			selectExpect: []string{"Estación Caltrain de San Francisco", "Gare Caltrain de San Francisco"},
		},
		{
			name:  "language and field_name",
			query: q,
			vars:  hw{"sha1": ctSha1, "where": hw{"language": "es", "field_name": "stop_name"}},
			sel: []testcaseSelector{
				{selector: "feed_versions.0.translations.#.translation", expect: []string{"Estación Caltrain de San Francisco", "Estación Caltrain de San José Diridon"}},
				{selector: `feed_versions.0.translations.#(field_value=="San Jose Diridon Caltrain")#.translation`, expect: []string{"Estación Caltrain de San José Diridon"}},
			},
		},
		{
			// BCP 47 language tags are case-insensitive.
			name:         "language in another case",
			query:        q,
			vars:         hw{"sha1": ctSha1, "where": hw{"language": "ES", "field_name": "stop_name"}},
			selector:     "feed_versions.0.translations.#.translation",
			selectExpect: []string{"Estación Caltrain de San Francisco", "Estación Caltrain de San José Diridon"},
		},
		{
			name:   "feed version without translations",
			query:  q,
			vars:   hw{"sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0"},
			expect: `{"feed_versions":[{"translations":[]}]}`,
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}
