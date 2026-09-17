package gql

import (
	"testing"

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

func TestPlaceResolver(t *testing.T) {
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
			name:         "ADM0",
			query:        q,
			vars:         hw{"level": "ADM0"},
			selector:     "places.#.adm0_name",
			selectExpect: []string{"United States of America"},
		},
		{
			name:         "ADM0 count",
			query:        q,
			vars:         hw{"level": "ADM0"},
			selector:     "places.#.count",
			selectExpect: []string{"4"},
		},
		{
			name:         "ADM0 where",
			query:        q,
			vars:         hw{"level": "ADM0", "where": hw{"adm0_name": "United States of America"}},
			selector:     "places.#.count",
			selectExpect: []string{"4"},
		},
		{
			name:         "ADM0 where 2",
			query:        q,
			vars:         hw{"level": "ADM0", "where": hw{"adm0_name": "Canada"}},
			selector:     "places.#.count",
			selectExpect: []string{},
		},
		{
			name:         "ADM0_ADM1",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1"},
			selector:     "places.#.adm1_name",
			selectExpect: []string{"California", "District of Columbia", "Florida", "Maryland", "Virginia"},
		},
		{
			name:         "ADM0_ADM1 count",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1"},
			selector:     "places.#.count",
			selectExpect: []string{"2", "1", "1", "1", "1"},
		},
		{
			name:         "ADM0_ADM1 where",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1", "where": hw{"adm1_name": "California"}},
			selector:     "places.#.count",
			selectExpect: []string{"2"},
		},
		{
			name:         "ADM0_ADM1_CITY",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1_CITY"},
			selector:     "places.#.city_name",
			selectExpect: []string{"Berkeley", "Oakland", "San Francisco", "San Jose", "San Mateo", "Tampa", "", "", "Washington,  D.C.", "Alexandria", "", "", ""},
		},
		{
			name:         "ADM0_ADM1_CITY where",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1_CITY", "where": hw{"city_name": "Oakland"}},
			selector:     "places.#.city_name",
			selectExpect: []string{"Oakland"},
		},
		// search
		{
			name:         "ADM0_ADM1_CITY search",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1_CITY", "where": hw{"search": "oak"}},
			selector:     "places.#.city_name",
			selectExpect: []string{"Oakland"},
		},
		{
			// selectExpect ignores order, so the order is checked directly: most
			// operators first, then by country, region and name.
			name:  "ADM0_ADM1_CITY search ordered by operator count",
			query: q,
			vars:  hw{"level": "ADM0_ADM1_CITY", "where": hw{"search": "san"}},
			f: func(t *testing.T, jj string) {
				assert.Equal(t, []string{"San Francisco", "San Mateo", "San Jose"}, cityNames(jj))
			},
		},
		{
			// San Mateo's association with BART ranks under 0.1, leaving it one operator.
			name:  "ADM0_ADM1_CITY search ordered by operator count at min_rank",
			query: q,
			vars:  hw{"level": "ADM0_ADM1_CITY", "where": hw{"search": "san", "min_rank": 0.1}},
			f: func(t *testing.T, jj string) {
				assert.Equal(t, []string{"San Francisco", "San Jose", "San Mateo"}, cityNames(jj))
			},
		},
		{
			name:         "ADM0_ADM1_CITY search requires every word",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1_CITY", "where": hw{"search": "san fran"}},
			selector:     "places.#.city_name",
			selectExpect: []string{"San Francisco"},
		},
		{
			name:         "ADM0_ADM1_CITY search qualified by region",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1_CITY", "where": hw{"search": "oakland california"}},
			selector:     "places.#.city_name",
			selectExpect: []string{"Oakland"},
		},
		{
			name:         "ADM0_ADM1_CITY search qualified by a region prefix",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1_CITY", "where": hw{"search": "san ca"}},
			selector:     "places.#.city_name",
			selectExpect: []string{"San Francisco", "San Jose", "San Mateo"},
		},
		{
			name:         "ADM0_ADM1_CITY search qualified by the wrong region",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1_CITY", "where": hw{"search": "oakland florida"}},
			selector:     "places.#.city_name",
			selectExpect: []string{},
		},
		{
			// A region alone is not a city: some word has to match the city's name.
			name:         "ADM0_ADM1_CITY search does not match the region",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1_CITY", "where": hw{"search": "california"}},
			selector:     "places.#.city_name",
			selectExpect: []string{},
		},
		{
			name:         "ADM0_ADM1_CITY search with punctuation as written",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1_CITY", "where": hw{"search": "Washington, D.C."}},
			selector:     "places.#.city_name",
			selectExpect: []string{"Washington,  D.C."},
		},
		{
			name:  "ADM1_CITY search",
			query: q,
			vars:  hw{"level": "ADM1_CITY", "where": hw{"search": "oak"}},
			sel: []testcaseSelector{
				{selector: "places.#.adm1_name", expect: []string{"California"}},
				{selector: "places.#.city_name", expect: []string{"Oakland"}},
			},
		},
		{
			name:         "ADM0_CITY search",
			query:        q,
			vars:         hw{"level": "ADM0_CITY", "where": hw{"search": "oak"}},
			selector:     "places.#.city_name",
			selectExpect: []string{"Oakland"},
		},
		{
			name:         "CITY search",
			query:        q,
			vars:         hw{"level": "CITY", "where": hw{"search": "oak"}},
			selector:     "places.#.city_name",
			selectExpect: []string{"Oakland"},
		},
		{
			name:         "ADM0_ADM1 search",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1", "where": hw{"search": "calif"}},
			selector:     "places.#.adm1_name",
			selectExpect: []string{"California"},
		},
		{
			name:         "ADM0_ADM1 search qualified by country",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1", "where": hw{"search": "virginia united states"}},
			selector:     "places.#.adm1_name",
			selectExpect: []string{"Virginia"},
		},
		{
			name:         "ADM0_ADM1 search does not match the country",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1", "where": hw{"search": "united"}},
			selector:     "places.#.adm1_name",
			selectExpect: []string{},
		},
		{
			name:         "ADM0_ADM1 search requires every word",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1", "where": hw{"search": "virginia florida"}},
			selector:     "places.#.adm1_name",
			selectExpect: []string{},
		},
		{
			name:         "ADM0 search",
			query:        q,
			vars:         hw{"level": "ADM0", "where": hw{"search": "united"}},
			selector:     "places.#.adm0_name",
			selectExpect: []string{"United States of America"},
		},
		{
			name:         "search with no level",
			query:        q,
			vars:         hw{"where": hw{"search": "united"}},
			selector:     "places.#.adm0_name",
			selectExpect: []string{"United States of America"},
		},
		{
			name:   "ADM0 search no match is an empty list",
			query:  q,
			vars:   hw{"level": "ADM0", "where": hw{"search": "canada"}},
			expect: `{"places":[]}`,
		},
		{
			name:         "search without usable words is ignored",
			query:        q,
			vars:         hw{"level": "ADM0", "where": hw{"search": "a b"}},
			selector:     "places.#.adm0_name",
			selectExpect: []string{"United States of America"},
		},
		// bbox
		{
			name:  "region bbox comes from the admin polygon",
			query: `query{ places(level: ADM0_ADM1, where: {adm1_name: "California"}) { bbox } }`,
			f: func(t *testing.T, jj string) {
				assert.InDelta(t, -124.409202, bboxMinLon(jj), 1e-5, "California reaches its own coastline")
			},
		},
		{
			// A country covers every region it contains, including those with no
			// operators — and the United States crosses the antimeridian at the
			// Aleutians, so its extent is only meaningful in the shifted frame.
			// Measured normally it reads -179.1 to 179.8, a useless 359 degrees.
			name:  "country bbox crossing the antimeridian",
			query: `query{ places(level: ADM0, where: {adm0_name: "United States of America"}) { bbox } }`,
			f: func(t *testing.T, jj string) {
				assert.InDelta(t, 172.476085, bboxMinLon(jj), 1e-5, "Attu, the westernmost Aleutian")
				assert.InDelta(t, 120.5, bboxWidth(jj), 0.1, "the real width, not 359")
			},
		},
		{
			// A city is a point in Natural Earth, buffered by the radius the place
			// association itself uses: about 0.46 degrees of longitude at this latitude.
			name:  "city bbox is buffered around the point",
			query: `query{ places(level: ADM0_ADM1_CITY, where: {city_name: "Oakland"}) { bbox } }`,
			f: func(t *testing.T, jj string) {
				assert.InDelta(t, -122.732419, bboxMinLon(jj), 1e-5, "Oakland widened by the association radius")
			},
		},
		{
			// A level that identifies a city by fewer names still gets a box.
			name:  "city bbox at a coarser level",
			query: `query{ places(level: ADM0_CITY, where: {city_name: "Oakland"}) { bbox } }`,
			f: func(t *testing.T, jj string) {
				assert.InDelta(t, -122.732419, bboxMinLon(jj), 1e-5, "same box as the finer level")
			},
		},
		// operators
		{
			name:         "ADM0 operators",
			query:        q,
			vars:         hw{"level": "ADM0"},
			selector:     "places.0.operators.#.onestop_id",
			selectExpect: []string{"o-dhv-hillsborougharearegionaltransit", "o-9q9-bayarearapidtransit", "o-9q9-caltrain", "o-dqcj-wmata"},
		},
		{
			name:         "ADM0_ADM1 operators",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1", "where": hw{"adm1_name": "California"}},
			selector:     "places.0.operators.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain"},
		},
		{
			name:         "ADM0_ADM1_CITY operators",
			query:        q,
			vars:         hw{"level": "ADM0_ADM1_CITY", "where": hw{"city_name": "Oakland"}},
			selector:     "places.0.operators.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}
