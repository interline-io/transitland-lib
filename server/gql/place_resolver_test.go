package gql

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/gql/internal/gqlcase"
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
			Name:  "city bbox is buffered around the point",
			Query: `query{ places(level: ADM0_ADM1_CITY, where: {city_name: "Oakland"}) { bbox } }`,
			F: func(t *testing.T, jj string) {
				assert.InDelta(t, -122.732419, bboxMinLon(jj), 1e-5, "Oakland widened by the association radius")
			},
		},
		{
			// A level that identifies a city by fewer names still gets a box.
			Name:  "city bbox at a coarser level",
			Query: `query{ places(level: ADM0_CITY, where: {city_name: "Oakland"}) { bbox } }`,
			F: func(t *testing.T, jj string) {
				assert.InDelta(t, -122.732419, bboxMinLon(jj), 1e-5, "same box as the finer level")
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
	c, _ := newTestClient(t)
	gqlcase.QueryTestcases(t, c, testcases)
}
