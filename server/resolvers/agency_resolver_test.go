package resolvers

import (
	"testing"

	"github.com/99designs/gqlgen/client"
)

func TestAgencyResolver(t *testing.T) {
	testcases := []testcase{
		{
			"basic",
			`query { agencies {agency_id}}`,
			hw{},
			``,
			"agencies.#.agency_id",
			[]string{"caltrain-ca-us", "BART"},
		},
		{
			"basic fields",
			`query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {onestop_id agency_id agency_name agency_lang agency_phone agency_timezone agency_url agency_email agency_fare_url feed_version_sha1 feed_onestop_id}}`,
			hw{"agency_id": "caltrain-ca-us"},
			`{"agencies":[{"agency_email":"","agency_fare_url":"","agency_id":"caltrain-ca-us","agency_lang":"en","agency_name":"Caltrain","agency_phone":"800-660-4287","agency_timezone":"America/Los_Angeles","agency_url":"http://www.caltrain.com","feed_onestop_id":"CT","feed_version_sha1":"d2813c293bcfd7a97dde599527ae6c62c98e66c6","onestop_id":"o-9q9-caltrain"}]}`,
			"",
			nil,
		},
		{
			"feed_version",
			`query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {feed_version { sha1 }}}`,
			hw{"agency_id": "caltrain-ca-us"},
			`{"agencies":[{"feed_version":{"sha1":"d2813c293bcfd7a97dde599527ae6c62c98e66c6"}}]}`,
			"",
			nil,
		},
		{
			"routes",
			`query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {routes { route_id }}}`, // todo: sorting
			hw{"agency_id": "caltrain-ca-us"},
			``,
			"agencies.0.routes.#.route_id",
			[]string{"Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130"},
		},
		{
			"places",
			`query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {places {adm0name adm1name name}}}`, // todo: sorting
			hw{"agency_id": "caltrain-ca-us"},
			``,
			"agencies.0.places.#.name",
			[]string{"San Mateo", "San Francisco", "San Jose", "", "Salinas"},
		},
		// TODO
		// {"census_geographies", }
	}
	c := client.New(newServer())
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			testquery(t, c, tc)
		})
	}
}
