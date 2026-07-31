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
		// bbox
		{
			name:  "region bbox comes from the admin polygon",
			query: `query{ places(level: ADM0_ADM1, where: {adm1_name: "California"}) { bbox } }`,
			f: func(t *testing.T, jj string) {
				assert.InDelta(t, -124.409202, bboxMinLon(jj), 1e-5, "California reaches its own coastline")
			},
		},
		{
			name:  "country bbox reaches every region",
			query: `query{ places(level: ADM0, where: {adm0_name: "United States of America"}) { bbox } }`,
			f: func(t *testing.T, jj string) {
				// Alaska, not California: a country covers every region it contains,
				// including those with no operators at all.
				assert.InDelta(t, -179.143503, bboxMinLon(jj), 1e-5, "country spans every region")
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
