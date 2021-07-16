package resolvers

import (
	"testing"
)

func TestAgencyResolver(t *testing.T) {
	vars := hw{"agency_id": "caltrain-ca-us"}
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
			vars,
			`{"agencies":[{"agency_email":"","agency_fare_url":"","agency_id":"caltrain-ca-us","agency_lang":"en","agency_name":"Caltrain","agency_phone":"800-660-4287","agency_timezone":"America/Los_Angeles","agency_url":"http://www.caltrain.com","feed_onestop_id":"CT","feed_version_sha1":"d2813c293bcfd7a97dde599527ae6c62c98e66c6","onestop_id":"o-9q9-caltrain"}]}`,
			"",
			nil,
		},
		{
			// just ensure this query completes successfully; checking coordinates is a pain and flaky.
			"geometry",
			`query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {geometry}}`,
			vars,
			``,
			"agencies.0.geometry.type",
			[]string{"Polygon"},
		},
		{
			"near 100m",
			`query {agencies(where:{near:{lat:-122.407974,lon:37.784471,radius:100.0}}) {agency_id}}`,
			hw{},
			``,
			"agencies.#.agency_id",
			[]string{"BART"},
		},
		{
			"near 10000m",
			`query {agencies(where:{near:{lat:-122.407974,lon:37.784471,radius:10000.0}}) {agency_id}}`,
			hw{},
			``,
			"agencies.#.agency_id",
			[]string{"caltrain-ca-us", "BART"},
		},
		{
			"within polygon",
			`query{agencies(where:{within:{type:"Polygon",coordinates:[[[-122.39803791046143,37.794626736533836],[-122.40106344223022,37.792303711508595],[-122.3965573310852,37.789641468930114],[-122.3938751220703,37.792354581451946],[-122.39803791046143,37.794626736533836]]]}}){agency_id}}`,
			hw{},
			``,
			"agencies.#.agency_id",
			[]string{"BART"},
		},
		{
			"within polygon big",
			`query{agencies(where:{within:{type:"Polygon",coordinates:[[[-122.39481925964355,37.80151060070086],[-122.41653442382812,37.78652126637423],[-122.39662170410156,37.76847577247014],[-122.37301826477051,37.784757615348575],[-122.39481925964355,37.80151060070086]]]}}){id agency_id}}`,
			hw{},
			``,
			"agencies.#.agency_id",
			[]string{"caltrain-ca-us", "BART"},
		},
		{
			"feed_version",
			`query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {feed_version { sha1 }}}`,
			vars,
			`{"agencies":[{"feed_version":{"sha1":"d2813c293bcfd7a97dde599527ae6c62c98e66c6"}}]}`,
			"",
			nil,
		},
		{
			"routes",
			`query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {routes { route_id }}}`,
			vars,
			``,
			"agencies.0.routes.#.route_id",
			[]string{"Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130"},
		},
		// places should test filters because it's not a root resolver
		{
			"places",
			`query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {places {adm0name adm1name name}}}`,
			vars,
			``,
			"agencies.0.places.#.name",
			[]string{"San Mateo", "San Francisco", "San Jose", "", "Salinas"},
		},
		{
			"places rank 0.25",
			`query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {places(where:{min_rank:0.25}) {adm0name adm1name name}}}`,
			vars,
			``,
			"agencies.0.places.#.name",
			[]string{"San Mateo", "San Jose", ""},
		},
		{
			"places rank 0.5",
			`query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {places(where:{min_rank:0.5}) {adm0name adm1name name}}}`,
			vars,
			``,
			"agencies.0.places.#.adm1name",
			[]string{"California"},
		},
		// search
		{
			"search",
			`query($search:String!) { agencies(where:{search:$search}) {agency_id}}`,
			hw{"search": "Bay Area"},
			``,
			"agencies.#.agency_id",
			[]string{"BART"},
		},
		{
			"search",
			`query($search:String!) { agencies(where:{search:$search}) {agency_id}}`,
			hw{"search": "caltrain"},
			``,
			"agencies.#.agency_id",
			[]string{"caltrain-ca-us"},
		},
		// TODO
		// {"census_geographies", }
	}
	c := newTestClient()
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			testquery(t, c, tc)
		})
	}
}
