package gqltest

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// bboxMinLon is the western edge of the first returned place's bounding box.
// ST_Extent emits the ring from its lower-left corner.
func bboxMinLon(jj string) float64 {
	return gjson.Get(jj, "places.0.bbox.coordinates.0.0.0").Float()
}

// cityNames is the returned places' city names, in order.
func cityNames(jj string) []string {
	names := []string{}
	for _, v := range gjson.Get(jj, "places.#.city_name").Array() {
		names = append(names, v.String())
	}
	return names
}

// bboxWidth is that box's span in degrees of longitude.
func bboxWidth(jj string) float64 {
	return gjson.Get(jj, "places.0.bbox.coordinates.0.2.0").Float() - bboxMinLon(jj)
}

func agencyResolverPlacesTestcases(t testing.TB, cfg model.Config) []testcase {
	vars := hw{"agency_id": "caltrain-ca-us"}
	testcases := []testcase{
		// places should test filters because it's not a root resolver
		{
			Name:         "places",
			Query:        `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {places {adm0_name adm0_iso adm1_name adm1_iso city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.0.places.#.city_name",
			SelectExpect: []string{"San Mateo", "San Francisco", "San Jose", ""},
		},
		{
			Name:         "places get adm0_iso",
			Query:        `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {places {adm0_name adm0_iso adm1_name adm1_iso city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.0.places.#.adm0_iso",
			SelectExpect: []string{"US", "US", "US", "US"},
		},
		{
			Name:         "places get adm1_iso",
			Query:        `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {places {adm0_name adm0_iso adm1_name adm1_iso city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.0.places.#.adm1_iso",
			SelectExpect: []string{"US-CA", "US-CA", "US-CA", "US-CA"},
		},
		{
			Name:         "places rank 0.25",
			Query:        `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {places(where:{min_rank:0.25}) {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.0.places.#.city_name",
			SelectExpect: []string{"San Mateo", "San Jose", ""},
		},
		{
			Name:         "places rank 0.75",
			Query:        `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {places(where:{min_rank:0.75}) {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.0.places.#.adm1_name",
			SelectExpect: []string{"California"},
		},
		// place iso codes
		{
			Name:         "places iso3166 country",
			Query:        `query { agencies(where:{adm0_iso: "US"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata"},
		},
		{
			Name:         "places iso3166 state",
			Query:        `query { agencies(where:{adm1_iso: "US-CA"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			Name:         "places iso3166 state lowercase",
			Query:        `query { agencies(where:{adm1_iso: "us-ca"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			Name:         "places iso3166 state and country",
			Query:        `query { agencies(where:{adm0_iso: "us", adm1_iso: "us-ca"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			Name:         "places iso3166 state and city",
			Query:        `query { agencies(where:{city_name: "oakland", adm1_iso: "us-ca"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			Name:         "places iso3166 state and city no result",
			Query:        `query { agencies(where:{city_name: "test", adm1_iso: "us-ca"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{},
		},
		{
			Name:         "places iso3166 state no results",
			Query:        `query { agencies(where:{adm1_iso: "US-NY"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{},
		},
		{
			Name:         "places state",
			Query:        `query { agencies(where:{adm1_name: "California"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			Name:         "places state no result",
			Query:        `query { agencies(where:{adm1_name: "New York"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{},
		},
		{
			// `%` is an ILIKE wildcard; unescaped it would match every adm0_iso.
			Name:         "places adm0_iso escapes ilike wildcards",
			Query:        `query { agencies(where:{adm0_iso: "%"}) {onestop_id}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{},
		},
		{
			Name:         "places city",
			Query:        `query { agencies(where:{city_name: "Berkeley"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			Name:         "places city 2",
			Query:        `query { agencies(where:{city_name: "San Jose"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{"o-9q9-caltrain"},
		},
		{
			Name:         "places city 2 lowercase",
			Query:        `query { agencies(where:{city_name: "san jose"}) {onestop_id places {adm0_name adm1_name city_name}}}`,
			Vars:         vars,
			Selector:     "agencies.#.onestop_id",
			SelectExpect: []string{"o-9q9-caltrain"},
		},
	}
	return testcases
}

func agencyResolverBatchedStopsTestcases(t testing.TB, cfg model.Config) []testcase {
	testcases := []testcase{
		{
			// Two agencies in one request must not share a limit.
			Name:  "batched agencies keep separate limits",
			Query: `query { agencies(where:{adm0_iso:"US"}) { onestop_id stops(limit:3) { stop_id } } }`,
			Sel: []testcaseSelector{
				{Selector: "agencies.#.onestop_id", Expect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata"}},
				{Selector: "agencies.0.stops.#.stop_id", ExpectCount: 3},
				{Selector: "agencies.3.stops.#.stop_id", ExpectCount: 3},
			},
		},
	}
	return testcases
}

func operatorResolverPlacesTestcases(t testing.TB, cfg model.Config) []testcase {
	testcases := []testcase{
		{
			Name:         "places iso3166 country",
			Query:        `query { operators(where:{adm0_iso: "US"}) {onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata"},
		},
		{
			Name:         "places iso3166 state",
			Query:        `query { operators(where:{adm1_iso: "US-CA"}) {onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			Name:         "places iso3166 state not found",
			Query:        `query { operators(where:{adm1_iso: "US-NY"}) {onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{},
		},
		{
			Name:         "places adm0_name",
			Query:        `query { operators(where:{adm0_name: "United States of America"}) {onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata"},
		},
		{
			Name:         "places adm1_name",
			Query:        `query { operators(where:{adm1_name: "California"}) {onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			Name:         "places adm1_name not found",
			Query:        `query { operators(where:{adm1_name: "Nowhere"}) {onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{},
		},
		{
			Name:         "places city_name",
			Query:        `query { operators(where:{city_name: "Oakland"}) {onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
	}
	return testcases
}

func placeResolverTestcases(t testing.TB, cfg model.Config) []testcase {
	q := `query($level: PlaceAggregationLevel,$where: PlaceFilter) {
		places(level: $level, where: $where) {
			adm0_name
			adm1_name
			city_name
			count
			operators {
				onestop_id
			}
		}
	}`
	testcases := []testcase{
		{
			Name:         "ADM0",
			Query:        q,
			Vars:         hw{"level": "ADM0"},
			Selector:     "places.#.adm0_name",
			SelectExpect: []string{"United States of America"},
		},
		{
			Name:         "ADM0 count",
			Query:        q,
			Vars:         hw{"level": "ADM0"},
			Selector:     "places.#.count",
			SelectExpect: []string{"4"},
		},
		{
			Name:         "ADM0 where",
			Query:        q,
			Vars:         hw{"level": "ADM0", "where": hw{"adm0_name": "United States of America"}},
			Selector:     "places.#.count",
			SelectExpect: []string{"4"},
		},
		{
			Name:         "ADM0 where 2",
			Query:        q,
			Vars:         hw{"level": "ADM0", "where": hw{"adm0_name": "Canada"}},
			Selector:     "places.#.count",
			SelectExpect: []string{},
		},
		{
			Name:         "ADM0_ADM1",
			Query:        q,
			Vars:         hw{"level": "ADM0_ADM1"},
			Selector:     "places.#.adm1_name",
			SelectExpect: []string{"California", "District of Columbia", "Florida", "Maryland", "Virginia"},
		},
		{
			Name:         "ADM0_ADM1 count",
			Query:        q,
			Vars:         hw{"level": "ADM0_ADM1"},
			Selector:     "places.#.count",
			SelectExpect: []string{"2", "1", "1", "1", "1"},
		},
		{
			Name:         "ADM0_ADM1 where",
			Query:        q,
			Vars:         hw{"level": "ADM0_ADM1", "where": hw{"adm1_name": "California"}},
			Selector:     "places.#.count",
			SelectExpect: []string{"2"},
		},
		{
			Name:         "ADM0_ADM1_CITY",
			Query:        q,
			Vars:         hw{"level": "ADM0_ADM1_CITY"},
			Selector:     "places.#.city_name",
			SelectExpect: []string{"Berkeley", "Oakland", "San Francisco", "San Jose", "San Mateo", "Tampa", "", "", "Washington,  D.C.", "Alexandria", "", "", ""},
		},
		{
			Name:         "ADM0_ADM1_CITY where",
			Query:        q,
			Vars:         hw{"level": "ADM0_ADM1_CITY", "where": hw{"city_name": "Oakland"}},
			Selector:     "places.#.city_name",
			SelectExpect: []string{"Oakland"},
		},
		// search
		{
			Name:         "ADM0_ADM1_CITY search",
			Query:        q,
			Vars:         hw{"level": "ADM0_ADM1_CITY", "where": hw{"search": "oak"}},
			Selector:     "places.#.city_name",
			SelectExpect: []string{"Oakland"},
		},
		{
			// selectExpect ignores order, so the order is checked directly: highest
			// count first, then by country, region and name.
			Name:  "ADM0_ADM1_CITY search ordered by count",
			Query: q,
			Vars:  hw{"level": "ADM0_ADM1_CITY", "where": hw{"search": "san"}},
			F: func(t *testing.T, jj string) {
				assert.Equal(t, []string{"San Francisco", "San Mateo", "San Jose"}, cityNames(jj))
			},
		},
		{
			// San Mateo's association with BART ranks under 0.1, leaving it one operator.
			Name:  "ADM0_ADM1_CITY search ordered by count at min_rank",
			Query: q,
			Vars:  hw{"level": "ADM0_ADM1_CITY", "where": hw{"search": "san", "min_rank": 0.1}},
			F: func(t *testing.T, jj string) {
				assert.Equal(t, []string{"San Francisco", "San Jose", "San Mateo"}, cityNames(jj))
			},
		},
		{
			Name:         "ADM0_ADM1_CITY search requires every word",
			Query:        q,
			Vars:         hw{"level": "ADM0_ADM1_CITY", "where": hw{"search": "san fran"}},
			Selector:     "places.#.city_name",
			SelectExpect: []string{"San Francisco"},
		},
		{
			Name:         "ADM0_ADM1_CITY search qualified by region",
			Query:        q,
			Vars:         hw{"level": "ADM0_ADM1_CITY", "where": hw{"search": "oakland california"}},
			Selector:     "places.#.city_name",
			SelectExpect: []string{"Oakland"},
		},
		{
			Name:         "ADM0_ADM1_CITY search qualified by a region prefix",
			Query:        q,
			Vars:         hw{"level": "ADM0_ADM1_CITY", "where": hw{"search": "san ca"}},
			Selector:     "places.#.city_name",
			SelectExpect: []string{"San Francisco", "San Jose", "San Mateo"},
		},
		{
			Name:         "ADM0_ADM1_CITY search qualified by the wrong region",
			Query:        q,
			Vars:         hw{"level": "ADM0_ADM1_CITY", "where": hw{"search": "oakland florida"}},
			Selector:     "places.#.city_name",
			SelectExpect: []string{},
		},
		{
			// A region alone is not a city: some word has to match the city's name.
			Name:         "ADM0_ADM1_CITY search does not match the region",
			Query:        q,
			Vars:         hw{"level": "ADM0_ADM1_CITY", "where": hw{"search": "california"}},
			Selector:     "places.#.city_name",
			SelectExpect: []string{},
		},
		{
			Name:         "ADM0_ADM1_CITY search with punctuation as written",
			Query:        q,
			Vars:         hw{"level": "ADM0_ADM1_CITY", "where": hw{"search": "Washington, D.C."}},
			Selector:     "places.#.city_name",
			SelectExpect: []string{"Washington,  D.C."},
		},
		{
			Name:  "ADM1_CITY search",
			Query: q,
			Vars:  hw{"level": "ADM1_CITY", "where": hw{"search": "oak"}},
			Sel: []testcaseSelector{
				{Selector: "places.#.adm1_name", Expect: []string{"California"}},
				{Selector: "places.#.city_name", Expect: []string{"Oakland"}},
			},
		},
		{
			Name:         "ADM0_CITY search",
			Query:        q,
			Vars:         hw{"level": "ADM0_CITY", "where": hw{"search": "oak"}},
			Selector:     "places.#.city_name",
			SelectExpect: []string{"Oakland"},
		},
		{
			Name:         "CITY search",
			Query:        q,
			Vars:         hw{"level": "CITY", "where": hw{"search": "oak"}},
			Selector:     "places.#.city_name",
			SelectExpect: []string{"Oakland"},
		},
		{
			Name:         "ADM0_ADM1 search",
			Query:        q,
			Vars:         hw{"level": "ADM0_ADM1", "where": hw{"search": "calif"}},
			Selector:     "places.#.adm1_name",
			SelectExpect: []string{"California"},
		},
		{
			Name:         "ADM0_ADM1 search qualified by country",
			Query:        q,
			Vars:         hw{"level": "ADM0_ADM1", "where": hw{"search": "virginia united states"}},
			Selector:     "places.#.adm1_name",
			SelectExpect: []string{"Virginia"},
		},
		{
			Name:         "ADM0_ADM1 search does not match the country",
			Query:        q,
			Vars:         hw{"level": "ADM0_ADM1", "where": hw{"search": "united"}},
			Selector:     "places.#.adm1_name",
			SelectExpect: []string{},
		},
		{
			Name:         "ADM0_ADM1 search requires every word",
			Query:        q,
			Vars:         hw{"level": "ADM0_ADM1", "where": hw{"search": "virginia florida"}},
			Selector:     "places.#.adm1_name",
			SelectExpect: []string{},
		},
		{
			Name:         "ADM0 search",
			Query:        q,
			Vars:         hw{"level": "ADM0", "where": hw{"search": "united"}},
			Selector:     "places.#.adm0_name",
			SelectExpect: []string{"United States of America"},
		},
		{
			Name:         "search with no level",
			Query:        q,
			Vars:         hw{"where": hw{"search": "united"}},
			Selector:     "places.#.adm0_name",
			SelectExpect: []string{"United States of America"},
		},
		{
			Name:   "ADM0 search no match is an empty list",
			Query:  q,
			Vars:   hw{"level": "ADM0", "where": hw{"search": "canada"}},
			Expect: `{"places":[]}`,
		},
		{
			Name:         "search without usable words is ignored",
			Query:        q,
			Vars:         hw{"level": "ADM0", "where": hw{"search": "a b"}},
			Selector:     "places.#.adm0_name",
			SelectExpect: []string{"United States of America"},
		},
		// bbox
		{
			Name:  "region bbox comes from the admin polygon",
			Query: `query{ places(level: ADM0_ADM1, where: {adm1_name: "California"}) { bbox } }`,
			F: func(t *testing.T, jj string) {
				assert.InDelta(t, -124.409202, bboxMinLon(jj), 1e-5, "California reaches its own coastline")
			},
		},
		{
			// A country covers every region it contains, including those with no
			// operators — and the United States crosses the antimeridian at the
			// Aleutians, so its extent is only meaningful in the shifted frame.
			// Measured normally it reads -179.1 to 179.8, a useless 359 degrees.
			Name:  "country bbox crossing the antimeridian",
			Query: `query{ places(level: ADM0, where: {adm0_name: "United States of America"}) { bbox } }`,
			F: func(t *testing.T, jj string) {
				assert.InDelta(t, 172.476085, bboxMinLon(jj), 1e-5, "Attu, the westernmost Aleutian")
				assert.InDelta(t, 120.5, bboxWidth(jj), 0.1, "the real width, not 359")
			},
		},
		{
			// A city is a point in Natural Earth, buffered by the radius the place
			// association itself uses: about 0.46 degrees of longitude at this latitude.
			// PostGIS buffers in a planar projection, and a buffer on the sphere lands
			// within 0.002 degrees of PostGIS's.
			Name:  "city bbox is buffered around the point",
			Query: `query{ places(level: ADM0_ADM1_CITY, where: {city_name: "Oakland"}) { bbox } }`,
			F: func(t *testing.T, jj string) {
				assert.InDelta(t, -122.732419, bboxMinLon(jj), 2e-3, "Oakland widened by the association radius")
			},
		},
		{
			// A level that identifies a city by fewer names still gets a box.
			Name:  "city bbox at a coarser level",
			Query: `query{ places(level: ADM0_CITY, where: {city_name: "Oakland"}) { bbox } }`,
			F: func(t *testing.T, jj string) {
				assert.InDelta(t, -122.732419, bboxMinLon(jj), 2e-3, "same box as the finer level")
			},
		},
		// operators
		{
			Name:         "ADM0 operators",
			Query:        q,
			Vars:         hw{"level": "ADM0"},
			Selector:     "places.0.operators.#.onestop_id",
			SelectExpect: []string{"o-dhv-hillsborougharearegionaltransit", "o-9q9-bayarearapidtransit", "o-9q9-caltrain", "o-dqcj-wmata"},
		},
		{
			Name:         "ADM0_ADM1 operators",
			Query:        q,
			Vars:         hw{"level": "ADM0_ADM1", "where": hw{"adm1_name": "California"}},
			Selector:     "places.0.operators.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			Name:         "ADM0_ADM1_CITY operators",
			Query:        q,
			Vars:         hw{"level": "ADM0_ADM1_CITY", "where": hw{"city_name": "Oakland"}},
			Selector:     "places.0.operators.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
	}
	return testcases
}

func stopResolverPlacesTestcases(t testing.TB, cfg model.Config) []testcase {
	q := `query($feed_version_sha1:String!, $stop_id:String!) { stops(where:{stop_id:$stop_id, feed_version_sha1:$feed_version_sha1}) { place { adm0_name adm1_name adm0_iso adm1_iso } } }`
	tcs := []testcase{
		{
			Name:         "usa",
			Query:        q,
			Vars:         hw{"feed_version_sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0", "stop_id": "FTVL"},
			Selector:     "stops.#.place.adm0_name",
			SelectExpect: []string{"United States of America"},
		},
		{
			Name:         "california",
			Query:        q,
			Vars:         hw{"feed_version_sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0", "stop_id": "FTVL"},
			Selector:     "stops.#.place.adm1_name",
			SelectExpect: []string{"California"},
		},
		{
			Name:         "adm0_iso",
			Query:        q,
			Vars:         hw{"feed_version_sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0", "stop_id": "FTVL"},
			Selector:     "stops.#.place.adm0_iso",
			SelectExpect: []string{"US"},
		},
		{
			Name:         "adm1_iso",
			Query:        q,
			Vars:         hw{"feed_version_sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0", "stop_id": "FTVL"},
			Selector:     "stops.#.place.adm1_iso",
			SelectExpect: []string{"US-CA"},
		},
		{
			Name:         "florida",
			Query:        q,
			Vars:         hw{"feed_version_sha1": "c969427f56d3a645195dd8365cde6d7feae7e99b", "stop_id": "8032"},
			Selector:     "stops.#.place.adm1_name",
			SelectExpect: []string{"Florida"},
		},
	}
	return tcs
}
