package resolvers

import (
	"testing"
)

func TestStopResolver(t *testing.T) {
	vars := hw{"stop_id": "MCAR"}
	testcases := []testcase{
		{
			"basic",
			`query($feed_version_sha1:String!) { stops(where:{feed_version_sha1:$feed_version_sha1}) { stop_id } }`, // just check BART
			hw{"feed_version_sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0"},
			``,
			"stops.#.stop_id",
			[]string{"12TH", "16TH", "19TH", "19TH_N", "24TH", "ANTC", "ASHB", "BALB", "BAYF", "CAST", "CIVC", "COLS", "COLM", "CONC", "DALY", "DBRK", "DUBL", "DELN", "PLZA", "EMBR", "FRMT", "FTVL", "GLEN", "HAYW", "LAFY", "LAKE", "MCAR", "MCAR_S", "MLBR", "MONT", "NBRK", "NCON", "OAKL", "ORIN", "PITT", "PCTR", "PHIL", "POWL", "RICH", "ROCK", "SBRN", "SFIA", "SANL", "SHAY", "SSAN", "UCTY", "WCRK", "WARM", "WDUB", "WOAK"},
		},
		{
			"basic fields",
			`query($stop_id: String!) {  stops(where:{stop_id:$stop_id}) {onestop_id feed_version_sha1 feed_onestop_id location_type stop_code stop_desc stop_id stop_name stop_timezone stop_url wheelchair_boarding zone_id} }`,
			vars,
			`{"stops":[{"feed_onestop_id":"BA","feed_version_sha1":"e535eb2b3b9ac3ef15d82c56575e914575e732e0","location_type":0,"onestop_id":"s-9q9p1wxf72-macarthur","stop_code":"","stop_desc":"","stop_id":"MCAR","stop_name":"MacArthur","stop_timezone":"","stop_url":"http://www.bart.gov/stations/MCAR/","wheelchair_boarding":1,"zone_id":"MCAR"}]}`,
			"",
			nil,
		},
		{
			// just ensure this query completes successfully; checking coordinates is a pain and flaky.
			"geometry",
			`query($stop_id: String!) {  stops(where:{stop_id:$stop_id}) {geometry} }`,
			vars,
			``,
			"stops.0.geometry.type",
			[]string{"Point"},
		},
		{
			"feed_version",
			`query($stop_id: String!) {  stops(where:{stop_id:$stop_id}) {feed_version_sha1} }`,
			vars,
			`{"stops":[{"feed_version_sha1":"e535eb2b3b9ac3ef15d82c56575e914575e732e0"}]}`,
			"",
			nil,
		},
		{
			"route_stops",
			`query($stop_id: String!) {  stops(where:{stop_id:$stop_id}) {route_stops{route{route_id route_short_name}}} }`,
			vars,
			``,
			"stops.0.route_stops.#.route.route_id",
			[]string{"01", "03", "07"},
		},
		{
			"where near 10m",
			`query {stops(where:{near:{lat:-122.407974,lon:37.784471,radius:10.0}}) {stop_id onestop_id geometry}}`,
			vars,
			``,
			"stops.#.stop_id",
			[]string{"POWL"},
		},
		{
			"where near 2000m",
			`query {stops(where:{near:{lat:-122.407974,lon:37.784471,radius:2000.0}}) {stop_id onestop_id geometry}}`,
			vars,
			``,
			"stops.#.stop_id",
			[]string{"70011", "70012", "CIVC", "EMBR", "MONT", "POWL"},
		},
		{
			"where within polygon",
			`query{stops(where:{within:{type:"Polygon",coordinates:[[[-122.396,37.8],[-122.408,37.79],[-122.393,37.778],[-122.38,37.787],[-122.396,37.8]]]}}){id stop_id}}`,
			hw{},
			``,
			"stops.#.stop_id",
			[]string{"EMBR", "MONT"},
		},
		{
			"where onestop_id",
			`query{stops(where:{onestop_id:"s-9q9k658fd1-sanjosediridoncaltrain"}) {stop_id} }`,
			vars,
			``,
			"stops.0.stop_id",
			[]string{"70262"},
		},
		{
			"where feed_version_sha1",
			`query($feed_version_sha1:String!) { stops(where:{feed_version_sha1:$feed_version_sha1}) { stop_id } }`, // just check BART
			hw{"feed_version_sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0"},
			``,
			"stops.#.stop_id",
			[]string{"12TH", "16TH", "19TH", "19TH_N", "24TH", "ANTC", "ASHB", "BALB", "BAYF", "CAST", "CIVC", "COLS", "COLM", "CONC", "DALY", "DBRK", "DUBL", "DELN", "PLZA", "EMBR", "FRMT", "FTVL", "GLEN", "HAYW", "LAFY", "LAKE", "MCAR", "MCAR_S", "MLBR", "MONT", "NBRK", "NCON", "OAKL", "ORIN", "PITT", "PCTR", "PHIL", "POWL", "RICH", "ROCK", "SBRN", "SFIA", "SANL", "SHAY", "SSAN", "UCTY", "WCRK", "WARM", "WDUB", "WOAK"},
		},
		{
			"where feed_onestop_id",
			`query{stops(where:{feed_onestop_id:"BA"}) { stop_id } }`, // just check BART
			hw{},
			``,
			"stops.#.stop_id",
			[]string{"12TH", "16TH", "19TH", "19TH_N", "24TH", "ANTC", "ASHB", "BALB", "BAYF", "CAST", "CIVC", "COLS", "COLM", "CONC", "DALY", "DBRK", "DUBL", "DELN", "PLZA", "EMBR", "FRMT", "FTVL", "GLEN", "HAYW", "LAFY", "LAKE", "MCAR", "MCAR_S", "MLBR", "MONT", "NBRK", "NCON", "OAKL", "ORIN", "PITT", "PCTR", "PHIL", "POWL", "RICH", "ROCK", "SBRN", "SFIA", "SANL", "SHAY", "SSAN", "UCTY", "WCRK", "WARM", "WDUB", "WOAK"},
		},
		{
			"where stop_id",
			`query{stops(where:{stop_id:"12TH"}) { stop_id } }`,
			hw{},
			``,
			"stops.#.stop_id",
			[]string{"12TH"},
		},
		{
			"where search",
			`query{stops(where:{search:"macarthur"}) { stop_id } }`,
			hw{},
			``,
			"stops.#.stop_id",
			[]string{"MCAR", "MCAR_S"},
		},
		{
			"where search 2",
			`query{stops(where:{search:"ftvl"}) { stop_id } }`,
			hw{},
			``,
			"stops.#.stop_id",
			[]string{"FTVL"},
		},
		{
			"where search 3",
			`query{stops(where:{search:"warm springs"}) { stop_id } }`,
			hw{},
			``,
			"stops.#.stop_id",
			[]string{"WARM"},
		},
		// TODO: parent, children; test data has no stations.
		// TODO: level, pathways_from_stop, pathways_to_stop: test data has no pathways...
		// TODO: census_geographies
		// stop_times
		{
			"stop_times",
			`query($stop_id: String!) {  stops(where:{stop_id:$stop_id}) {stop_times { trip { trip_id} }} }`,
			hw{"stop_id": "70302"}, // Morgan hill
			``,
			"stops.0.stop_times.#.trip.trip_id",
			[]string{"268", "274", "156"},
		},
		{
			"stop_times where weekday_morning",
			`query($stop_id: String!, $service_date:Date!) {  stops(where:{stop_id:$stop_id}) {stop_times(where:{service_date:$service_date, start_time:21600, end_time:25200}) { trip { trip_id} }} }`,
			hw{"stop_id": "MCAR", "service_date": "2018-05-29"},
			``,
			"stops.0.stop_times.#.trip.trip_id",
			[]string{"3830503WKDY", "3850526WKDY", "3610541WKDY", "3630556WKDY", "3650611WKDY", "2210533WKDY", "2230548WKDY", "2250603WKDY", "2270618WKDY", "4410518WKDY", "4430533WKDY", "4450548WKDY", "4470603WKDY"},
		},
		{
			"stop_times where sunday_morning",
			`query($stop_id: String!, $service_date:Date!) {  stops(where:{stop_id:$stop_id}) {stop_times(where:{service_date:$service_date, start_time:21600, end_time:36000}) { trip { trip_id} }} }`,
			hw{"stop_id": "MCAR", "service_date": "2018-05-27"},
			``,
			"stops.0.stop_times.#.trip.trip_id",
			[]string{"3730756SUN", "3750757SUN", "3770801SUN", "3790821SUN", "3610841SUN", "3630901SUN", "2230800SUN", "2250748SUN", "2270808SUN", "2290828SUN", "2310848SUN", "2330908SUN"},
		},
		{
			"stop_times where saturday_evening",
			`query($stop_id: String!, $service_date:Date!) {  stops(where:{stop_id:$stop_id}) {stop_times(where:{service_date:$service_date, start_time:57600, end_time:72000}) { trip { trip_id} }} }`,
			hw{"stop_id": "MCAR", "service_date": "2018-05-26"},
			``,
			"stops.0.stop_times.#.trip.trip_id",
			[]string{"3611521SAT", "3631541SAT", "3651601SAT", "3671621SAT", "3691641SAT", "3711701SAT", "3731721SAT", "3751741SAT", "3771801SAT", "3791821SAT", "3611841SAT", "3631901SAT", "2231528SAT", "2251548SAT", "2271608SAT", "2291628SAT", "2311648SAT", "2331708SAT", "2351728SAT", "2211748SAT", "2231808SAT", "2251828SAT", "2271848SAT", "2291908SAT", "4471533SAT", "4491553SAT", "4511613SAT", "4531633SAT", "4411653SAT", "4431713SAT", "4451733SAT", "4471753SAT", "4491813SAT", "4511833SAT", "4531853SAT"},
		},
		// TODO: census_geographies
		// TODO: route_stop_buffer
	}
	c := newTestClient()
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			testquery(t, c, tc)
		})
	}
}
