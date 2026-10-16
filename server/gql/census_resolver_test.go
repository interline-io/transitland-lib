package gql

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestCensusResolver(t *testing.T) {
	c, cfg := newTestClient(t)
	// Define checks and get IDs for tests
	countyArea := 2126920949.9718297 // Area of Alameda County in m^2
	geographyId := 0
	if err := cfg.Adapter.DBX().QueryRowx(`select id from tl_census_geographies where geoid = '1400000US06001403000'`).Scan(&geographyId); err != nil {
		t.Errorf("could not get geography id for test: %s", err.Error())
	}
	bartFtvlStopId := 0
	if err := cfg.Adapter.DBX().QueryRowx(`select gtfs_stops.id from gtfs_stops join feed_states using(feed_version_id) where stop_id = 'FTVL'`).Scan(&bartFtvlStopId); err != nil {
		t.Errorf("could not get stop id for test: %s", err.Error())
	}
	bartMcarStopId := 0
	if err := cfg.Adapter.DBX().QueryRowx(`select gtfs_stops.id from gtfs_stops join feed_states using(feed_version_id) where stop_id = 'MCAR'`).Scan(&bartMcarStopId); err != nil {
		t.Errorf("could not get stop id for test: %s", err.Error())
	}
	geographyIdFor := func(geoid string) int {
		id := 0
		if err := cfg.Adapter.DBX().QueryRowx(`select id from tl_census_geographies where geoid = $1`, geoid).Scan(&id); err != nil {
			t.Errorf("could not get geography id for %s: %s", geoid, err.Error())
		}
		return id
	}
	alamedaCountyId := geographyIdFor("0500000US06001")
	kingCountyId := geographyIdFor("0500000US53033")
	ftvlTractId := geographyIdFor("1400000US06001406100") // Contains the FTVL stop

	// Search areas with more vertices than the finder intersects whole, so
	// they are subdivided: a 600-vertex circle inside Alameda County, and the
	// 2,961-vertex outline of tract 450701.
	var circleRing [][]float64
	for i := 0; i <= 600; i++ {
		a := 2 * math.Pi * float64(i%600) / 600
		circleRing = append(circleRing, []float64{-122.25 + 0.03*math.Cos(a), 37.80 + 0.03*math.Sin(a)})
	}
	circle := hw{"type": "Polygon", "coordinates": [][][]float64{circleRing}}
	circleArea := 27638158.48917035 // Derived by direct SQL, without subdivision
	tractOutline := hw{}
	var tractOutlineJson string
	if err := cfg.Adapter.DBX().QueryRowx(`select ST_AsGeoJSON(ST_GeometryN(geometry::geometry, 1)) from tl_census_geographies where geoid = '1400000US06001450701'`).Scan(&tractOutlineJson); err != nil {
		t.Errorf("could not get tract outline for test: %s", err.Error())
	}
	if err := json.Unmarshal([]byte(tractOutlineJson), &tractOutline); err != nil {
		t.Errorf("could not parse tract outline for test: %s", err.Error())
	}

	// Define test cases
	vars := hw{}
	testcases := []testcase{
		// Datasets
		{
			name:   "dataset basic fields",
			query:  `query { census_datasets {name} }`,
			vars:   vars,
			expect: `{"census_datasets":[{"name":"acsdt5y2022"},{"name":"tiger2024"}]}`,
		},
		{
			name:   "dataset filter by name",
			query:  `query { census_datasets(where:{name:"acsdt5y2022"}) {name} }`,
			vars:   vars,
			expect: `{"census_datasets":[{"name":"acsdt5y2022"}]}`,
		},
		{
			name:   "dataset filter by search",
			query:  `query { census_datasets(where:{search:"tiger"}) {name } }`,
			vars:   vars,
			expect: `{"census_datasets":[{"name":"tiger2024"}]}`,
		},
		// Dataset layers
		{
			name:   "dataset layers",
			query:  `query { census_datasets(where:{name:"tiger2024"}) {name layers { name description }} }`,
			vars:   vars,
			expect: `{"census_datasets":[{"layers":[{"description":"Layer: uac20","name":"uac20"},{"description":"Layer: cbsa","name":"cbsa"},{"description":"Layer: csa","name":"csa"},{"description":"Layer: state","name":"state"},{"description":"Layer: county","name":"county"},{"description":"Layer: place","name":"place"},{"description":"Layer: tract","name":"tract"}],"name":"tiger2024"}]}`,
		},
		{
			name:   "dataset layer geographies",
			query:  `query { census_datasets(where:{name:"tiger2024"}) {name layers { name geographies(where:{search:"ala"}) { name } }} }`,
			vars:   vars,
			expect: `{"census_datasets":[{"layers":[{"geographies":null,"name":"uac20"},{"geographies":null,"name":"cbsa"},{"geographies":null,"name":"csa"},{"geographies":null,"name":"state"},{"geographies":[{"name":"Alameda"}],"name":"county"},{"geographies":[{"name":"Acalanes Ridge"}],"name":"place"},{"geographies":null,"name":"tract"}],"name":"tiger2024"}]}`,
		},
		// Dataset Geographies
		{
			name:              "dataset geographies",
			query:             `query { census_datasets(where:{name:"tiger2024"}) {name geographies(limit:5) { geoid }} }`,
			vars:              vars,
			selector:          "census_datasets.0.geographies.#.geoid",
			selectExpectCount: 5,
		},
		{
			name:         "dataset geographies with layer",
			query:        `query { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer:"county"}) { name geoid }} }`,
			vars:         vars,
			selector:     "census_datasets.0.geographies.#.name",
			selectExpect: []string{"King", "Alameda"},
		},
		{
			name:   "dataset geographies with layer and adm names",
			query:  `query { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer:"county"}) { name geoid adm0_name adm1_name adm0_iso adm1_iso }} }`,
			vars:   vars,
			expect: `{"census_datasets":[{"name":"tiger2024","geographies":[{"adm0_iso":"US","adm0_name":"United States","adm1_iso":"US-WA","adm1_name":"Washington","geoid":"0500000US53033","name":"King"},{"adm0_iso":"US","adm0_name":"United States","adm1_iso":"US-CA","adm1_name":"California","geoid":"0500000US06001","name":"Alameda"}]}]}`,
		},
		{
			name:         "dataset geographies are multipolygon",
			query:        `query { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{search:"king"}) { name geoid geometry }} }`,
			vars:         vars,
			selector:     "census_datasets.0.geographies.#.geometry.type",
			selectExpect: []string{"MultiPolygon"},
		},
		{
			name:         "dataset geographies with search",
			query:        `query { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{search:"king"}) { name geoid }} }`,
			vars:         vars,
			selector:     "census_datasets.0.geographies.#.geoid",
			selectExpect: []string{"0500000US53033"},
		},
		{
			name:         "dataset geographies with search and layer",
			query:        `query { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", search:"288.02"}) { name geoid }} }`,
			vars:         vars,
			selector:     "census_datasets.0.geographies.#.geoid",
			selectExpect: []string{"1400000US53033028802"},
		},
		{
			name:         "dataset geographies near point 1",
			query:        `query { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", location:{near: {lon:-122.270, lat:37.805, radius:1000}}}) { name geoid }} }`,
			vars:         vars,
			selector:     "census_datasets.0.geographies.#.geoid",
			selectExpect: []string{"1400000US06001403501", "1400000US06001403402", "1400000US06001403302", "1400000US06001402802", "1400000US06001403301", "1400000US06001403401", "1400000US06001402801", "1400000US06001401400", "1400000US06001403000", "1400000US06001402600", "1400000US06001403100", "1400000US06001401300", "1400000US06001402900", "1400000US06001401600", "1400000US06001402700", "1400000US06001983200", "1400000US06001403701"},
		},
		{
			name:         "dataset geographies near point 2",
			query:        `query { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", location:{near: {lon:-122.270, lat:37.805, radius:100}}}) { name geoid }} }`,
			vars:         vars,
			selector:     "census_datasets.0.geographies.#.geoid",
			selectExpect: []string{"1400000US06001402801", "1400000US06001402900"},
		},
		{
			name:         "dataset geographies near point 3",
			query:        `query { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", location:{near: {lon:-122.270, lat:37.805, radius:10}}}) { name geoid }} }`,
			vars:         vars,
			selector:     "census_datasets.0.geographies.#.geoid",
			selectExpect: []string{"1400000US06001402900"},
		},
		{
			name:         "dataset geographies near point 4",
			query:        `query { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "county", location:{near: {lon:-122.270, lat:37.805, radius:1000}}}) { name geoid }} }`,
			vars:         vars,
			selector:     "census_datasets.0.geographies.#.geoid",
			selectExpect: []string{"0500000US06001"},
		},
		{
			name:         "dataset geographies in bbox 1",
			query:        `query($bbox:BoundingBox) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", location:{bbox:$bbox}}) { name geoid }} }`,
			vars:         hw{"bbox": hw{"min_lon": -122.27187746297761, "min_lat": 37.86760085920619, "max_lon": -122.26331772424285, "max_lat": 37.874244507564896}},
			selector:     "census_datasets.0.geographies.#.geoid",
			selectExpect: []string{"1400000US06001982100", "1400000US06001422902", "1400000US06001422901", "1400000US06001422400", "1400000US06001422800"},
		},
		{
			name:         "dataset geographies in bbox 2",
			query:        `query($bbox:BoundingBox) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", location:{bbox:$bbox}}) { name geoid }} }`,
			vars:         hw{"bbox": hw{"min_lon": -122.2698781543005, "min_lat": 37.80700393130445, "max_lon": -122.2677640139239, "max_lat": 37.8088734037938}},
			selector:     "census_datasets.0.geographies.#.geoid",
			selectExpect: []string{"1400000US06001402801", "1400000US06001402900"},
		},
		{
			name:         "dataset geographies by id",
			query:        `query($ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{ids:$ids}) { name geoid }} }`,
			vars:         hw{"ids": []int{geographyId}},
			selector:     "census_datasets.0.geographies.#.geoid",
			selectExpect: []string{"1400000US06001403000"},
		},
		{
			name:   "dataset geographies with focus",
			query:  `query($focus: FocusPoint) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer:"county", location:{focus:$focus}}) { name geoid }} }`,
			vars:   hw{"focus": hw{"lon": -122.270, "lat": 37.805}},
			expect: `{"census_datasets":[{"name":"tiger2024","geographies":[{"geoid":"0500000US06001","name":"Alameda"},{"geoid":"0500000US53033","name":"King"}]}]}`,
		},
		{
			name:   "dataset geographies with focus 2",
			query:  `query($focus: FocusPoint) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer:"county", location:{focus:$focus}}) { name geoid }} }`,
			vars:   hw{"focus": hw{"lon": -122.180, "lat": 48.390}},
			expect: `{"census_datasets":[{"name":"tiger2024","geographies":[{"geoid":"0500000US53033","name":"King"},{"geoid":"0500000US06001","name":"Alameda"}]}]}`,
		},
		{
			name:   "dataset geographies layer",
			query:  `query($ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{ids:$ids}) { name geoid layer { name }}} }`,
			vars:   hw{"ids": []int{geographyId}},
			expect: `{"census_datasets":[{"geographies":[{"geoid":"1400000US06001403000","layer":{"name":"tract"},"name":"4030"}],"name":"tiger2024"}]}`,
		},
		{
			name:   "dataset geographies source",
			query:  `query($ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{ids:$ids}) { name geoid source { name }}} }`,
			vars:   hw{"ids": []int{geographyId}},
			expect: `{"census_datasets":[{"geographies":[{"geoid":"1400000US06001403000","name":"4030","source":{"name":"tl_2024_06_tract.zip"}}],"name":"tiger2024"}]}`,
		},
		// Sources
		{
			name:   "sources",
			query:  `query { census_datasets(where:{name:"acsdt5y2022"}) {name sources { name }} }`,
			vars:   vars,
			expect: ` {"census_datasets":[{"name":"acsdt5y2022","sources":[{"name":"acsdt5y2022-b01001.dat"},{"name":"acsdt5y2022-b01001a.dat"},{"name":"acsdt5y2022-b01001b.dat"},{"name":"acsdt5y2022-b01001c.dat"},{"name":"acsdt5y2022-b01001d.dat"},{"name":"acsdt5y2022-b01001e.dat"},{"name":"acsdt5y2022-b01001f.dat"},{"name":"acsdt5y2022-b01001g.dat"},{"name":"acsdt5y2022-b01001h.dat"},{"name":"acsdt5y2022-b01001i.dat"}]}]}`,
		},
		// Source layers
		{
			name:   "source layers",
			query:  `query { census_datasets(where:{name:"tiger2024"}) {name sources(where:{name:"tl_2024_06_tract.zip"}) {name layers { name description }} } }`,
			vars:   vars,
			expect: `{"census_datasets":[{"name":"tiger2024","sources":[{"layers":[{"description":"Layer: tract","name":"tract"}],"name":"tl_2024_06_tract.zip"}]}]}`,
		},
		// Source tables: an ACS source resolves to the table it loaded; a TIGER source
		// (geographies only) resolves to none -- the tables-vs-layers distinction.
		{
			name:   "source tables",
			query:  `query { census_datasets(where:{name:"acsdt5y2022"}) {name sources(where:{name:"acsdt5y2022-b01001.dat"}) {name tables { table_name }} } }`,
			vars:   vars,
			expect: `{"census_datasets":[{"name":"acsdt5y2022","sources":[{"name":"acsdt5y2022-b01001.dat","tables":[{"table_name":"b01001"}]}]}]}`,
		},
		{
			name:   "source tables empty for tiger",
			query:  `query { census_datasets(where:{name:"tiger2024"}) {name sources(where:{name:"tl_2024_06_tract.zip"}) {name tables { table_name }} } }`,
			vars:   vars,
			expect: `{"census_datasets":[{"name":"tiger2024","sources":[{"name":"tl_2024_06_tract.zip","tables":null}]}]}`,
		},
		{
			name:   "source geographies",
			query:  `query { census_datasets(where:{name:"tiger2024"}) {name sources(where:{name:"tl_2024_us_county.zip"}) {name geographies(where:{search:"ala"}) { name } } } }`,
			vars:   vars,
			expect: `{"census_datasets":[{"name":"tiger2024","sources":[{"geographies":[{"name":"Alameda"}],"name":"tl_2024_us_county.zip"}]}]}`,
		},
		// Intersection areas
		{
			name:  "dataset intersection areas by stop buffer - tract",
			query: `query($stop_ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", location:{stop_buffer:{stop_ids:$stop_ids, radius:100}}}) { name geoid geometry_area geometry intersection_geometry intersection_area }} }`,
			vars:  hw{"stop_ids": []int{bartFtvlStopId}},
			f: func(t *testing.T, jj string) {
				testIntersectionArea(
					t,
					gjson.Get(jj, "census_datasets.0.geographies").Array(),
					1,
					1918913.384123061,
					31235.844686623335,
				)
			},
		},
		{
			name:  "dataset intersection areas by stop buffer - tract, 1km",
			query: `query($stop_ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) {name geographies(limit:1000, where:{layer: "tract", location:{stop_buffer:{stop_ids:$stop_ids, radius:1000}}}) { name geoid geometry_area intersection_geometry intersection_area }} }`,
			vars:  hw{"stop_ids": []int{bartFtvlStopId}},
			f: func(t *testing.T, jj string) {
				testIntersectionArea(
					t,
					gjson.Get(jj, "census_datasets.0.geographies").Array(),
					10,
					9496243.013759723,
					3123584.4581301054,
				)
			},
		},
		{
			// Radius at the clamp ceiling (1600m ~ 1 mile). Expected values
			// derived independently via direct SQL; under the previous 1000m
			// clamp this query would silently return the 1km results instead.
			name:  "dataset intersection areas by stop buffer - tract, 1600m clamp ceiling",
			query: `query($stop_ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) {name geographies(limit:1000, where:{layer: "tract", location:{stop_buffer:{stop_ids:$stop_ids, radius:1600}}}) { name geoid geometry_area intersection_geometry intersection_area }} }`,
			vars:  hw{"stop_ids": []int{bartFtvlStopId}},
			f: func(t *testing.T, jj string) {
				testIntersectionArea(
					t,
					gjson.Get(jj, "census_datasets.0.geographies").Array(),
					16,
					17083972.98913558,
					7996376.297622201,
				)
			},
		},
		{
			name:  "dataset intersection areas by stop buffer, 2 stops - tract",
			query: `query($stop_ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", location:{stop_buffer:{stop_ids:$stop_ids, radius:100}}}) { name geoid geometry_area intersection_geometry intersection_area }} }`,
			vars:  hw{"stop_ids": []int{bartFtvlStopId, bartMcarStopId}},
			f: func(t *testing.T, jj string) {
				testIntersectionArea(
					t,
					gjson.Get(jj, "census_datasets.0.geographies").Array(),
					3,
					3941244.1183623667,
					62472.080173725684,
				)
			},
		},
		{
			name:  "dataset intersection areas within feature - tract",
			query: `query($feature:Polygon) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", location:{within:$feature}}) { name geoid geometry_area intersection_geometry intersection_area }} }`,
			vars: hw{"feature": hw{"type": "Polygon", "coordinates": [][][]float64{{
				{-122.27463277683867, 37.805635064682264},
				{-122.28006473340696, 37.80461858815316},
				{-122.27406099193678, 37.801456127261474},
				{-122.2754189810789, 37.79671218203016},
				{-122.27041586318674, 37.799648945955155},
				{-122.26398328303992, 37.79863238703946},
				{-122.26791430424078, 37.80247264731531},
				{-122.26441212171653, 37.80693387544258},
				{-122.269558185834, 37.806199767818995},
				{-122.27313184147101, 37.81066077079923},
				{-122.27463277683867, 37.805635064682264},
			}}}},
			f: func(t *testing.T, jj string) {
				testIntersectionArea(
					t,
					gjson.Get(jj, "census_datasets.0.geographies").Array(),
					11,
					4755616.146625203,
					829386.4530724211,
				)
			},
		},
		{
			name:  "dataset intersection areas within feature - county",
			query: `query($feature:Polygon) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "county", location:{within:$feature}}) { name geoid geometry_area intersection_geometry intersection_area }} }`,
			vars: hw{"feature": hw{"type": "Polygon", "coordinates": [][][]float64{{
				{-122.27463277683867, 37.805635064682264},
				{-122.28006473340696, 37.80461858815316},
				{-122.27406099193678, 37.801456127261474},
				{-122.2754189810789, 37.79671218203016},
				{-122.27041586318674, 37.799648945955155},
				{-122.26398328303992, 37.79863238703946},
				{-122.26791430424078, 37.80247264731531},
				{-122.26441212171653, 37.80693387544258},
				{-122.269558185834, 37.806199767818995},
				{-122.27313184147101, 37.81066077079923},
				{-122.27463277683867, 37.805635064682264},
			}}}},
			f: func(t *testing.T, jj string) {
				testIntersectionArea(
					t,
					gjson.Get(jj, "census_datasets.0.geographies").Array(),
					1,
					countyArea,
					829386.6022729833,
				)
			},
		},
		{
			name:  "dataset intersection areas within feature - county (big)",
			query: `query($feature:Polygon) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "county", location:{within:$feature}}) { name geoid geometry_area intersection_geometry intersection_area }} }`,
			vars: hw{"feature": hw{"type": "Polygon", "coordinates": [][][]float64{{
				{-123.77489413290716, 38.794161309061735},
				{-122.69431950796763, 35.52679604934255},
				{-119.9104881819854, 37.991860068760204},
				{-123.77489413290716, 38.794161309061735},
			}}}},
			f: func(t *testing.T, jj string) {
				testIntersectionArea(
					t,
					gjson.Get(jj, "census_datasets.0.geographies").Array(),
					1,
					countyArea,
					countyArea,
				)
			},
		},
		{
			// Combined within + stop_buffer must honor BOTH filters. Here
			// `within` is a large polygon that fully contains the FTVL buffer,
			// so the search area reduces to the buffer and the result matches
			// the stop-buffer-only case (one tract, ~31235 m^2 of buffer area).
			// Before the combined branch, `within` won outright and the buffer
			// was ignored — this query would have returned every tract in the
			// polygon instead.
			name:  "combined within and stop buffer - tract",
			query: `query($feature:Polygon, $stop_ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", location:{within:$feature, stop_buffer:{stop_ids:$stop_ids, radius:100}}}) { name geoid geometry_area intersection_geometry intersection_area }} }`,
			vars: hw{
				"stop_ids": []int{bartFtvlStopId},
				"feature": hw{"type": "Polygon", "coordinates": [][][]float64{{
					{-123.77489413290716, 38.794161309061735},
					{-122.69431950796763, 35.52679604934255},
					{-119.9104881819854, 37.991860068760204},
					{-123.77489413290716, 38.794161309061735},
				}}},
			},
			f: func(t *testing.T, jj string) {
				testIntersectionArea(
					t,
					gjson.Get(jj, "census_datasets.0.geographies").Array(),
					1,
					1918913.384123061,
					31235.844686623343,
				)
			},
		},
		{
			// Partial overlap: the polygon's eastern edge runs through the
			// FTVL stop, so the clip is the west half of the 100m buffer and
			// the intersection area is half the buffer-only case. This is the
			// case that pins ST_Intersection semantics from both sides: if
			// `within` were ignored the result would be the full 31235.84, and
			// if the buffer were ignored it would be every tract in the
			// rectangle.
			name:  "combined within and stop buffer partial overlap - tract",
			query: `query($feature:Polygon, $stop_ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", location:{within:$feature, stop_buffer:{stop_ids:$stop_ids, radius:100}}}) { name geoid geometry_area intersection_geometry intersection_area }} }`,
			vars: hw{
				"stop_ids": []int{bartFtvlStopId},
				"feature": hw{"type": "Polygon", "coordinates": [][][]float64{{
					{-122.5, 37.5},
					{-122.224175, 37.5},
					{-122.224175, 38.0},
					{-122.5, 38.0},
					{-122.5, 37.5},
				}}},
			},
			f: func(t *testing.T, jj string) {
				testIntersectionArea(
					t,
					gjson.Get(jj, "census_datasets.0.geographies").Array(),
					1,
					1918913.384123061,
					15617.923991315816,
				)
			},
		},
		{
			// Disjoint polygon and buffer: the clip intersection is empty and
			// no geographies match, rather than erroring or falling back to
			// either filter alone.
			name:  "combined within and stop buffer disjoint - tract",
			query: `query($feature:Polygon, $stop_ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", location:{within:$feature, stop_buffer:{stop_ids:$stop_ids, radius:100}}}) { name geoid intersection_area }} }`,
			vars: hw{
				"stop_ids": []int{bartFtvlStopId},
				"feature": hw{"type": "Polygon", "coordinates": [][][]float64{{
					{-121.6, 38.4},
					{-121.4, 38.4},
					{-121.4, 38.7},
					{-121.6, 38.7},
					{-121.6, 38.4},
				}}},
			},
			f: func(t *testing.T, jj string) {
				assert.Empty(t, gjson.Get(jj, "census_datasets.0.geographies").Array(), "disjoint clip must match no geographies")
			},
		},
		{
			// Radius 0 combines as polygon-restricted point matching: the FTVL
			// stop point is inside the polygon, so its containing tract
			// matches.
			name:  "combined within and stop buffer radius 0 - tract",
			query: `query($feature:Polygon, $stop_ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", location:{within:$feature, stop_buffer:{stop_ids:$stop_ids, radius:0}}}) { name geoid }} }`,
			vars: hw{
				"stop_ids": []int{bartFtvlStopId},
				"feature": hw{"type": "Polygon", "coordinates": [][][]float64{{
					{-123.77489413290716, 38.794161309061735},
					{-122.69431950796763, 35.52679604934255},
					{-119.9104881819854, 37.991860068760204},
					{-123.77489413290716, 38.794161309061735},
				}}},
			},
			expect: `{"census_datasets":[{"name":"tiger2024","geographies":[{"geoid":"1400000US06001406100","name":"4061"}]}]}`,
		},
		{
			// Radius 0 with the stop outside the polygon: the point is
			// filtered out, not silently matched.
			name:  "combined within and stop buffer radius 0 outside polygon - tract",
			query: `query($feature:Polygon, $stop_ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", location:{within:$feature, stop_buffer:{stop_ids:$stop_ids, radius:0}}}) { name geoid }} }`,
			vars: hw{
				"stop_ids": []int{bartFtvlStopId},
				"feature": hw{"type": "Polygon", "coordinates": [][][]float64{{
					{-121.6, 38.4},
					{-121.4, 38.4},
					{-121.4, 38.7},
					{-121.6, 38.7},
					{-121.6, 38.4},
				}}},
			},
			f: func(t *testing.T, jj string) {
				assert.Empty(t, gjson.Get(jj, "census_datasets.0.geographies").Array(), "stop outside the polygon must not match")
			},
		},
		{
			// A bbox containing the whole buffer reduces to the
			// stop-buffer-only result, the same as the `within` case above.
			// Before bbox composed, bbox won outright and the buffer was
			// ignored — this would have returned every tract in the box.
			name:  "combined bbox and stop buffer - tract",
			query: `query($bbox:BoundingBox, $stop_ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", location:{bbox:$bbox, stop_buffer:{stop_ids:$stop_ids, radius:100}}}) { name geoid geometry_area intersection_area }} }`,
			vars: hw{
				"stop_ids": []int{bartFtvlStopId},
				"bbox":     hw{"min_lon": -123.0, "min_lat": 37.0, "max_lon": -122.0, "max_lat": 38.0},
			},
			f: func(t *testing.T, jj string) {
				testIntersectionArea(
					t,
					gjson.Get(jj, "census_datasets.0.geographies").Array(),
					1,
					1918913.384123061,
					31235.844686623343,
				)
			},
		},
		{
			// Same rectangle as the `within` partial-overlap case, expressed
			// as a bbox: its eastern edge runs through the FTVL stop, so the
			// clip is the west half of the buffer. ST_MakeEnvelope and the
			// equivalent polygon must therefore produce the identical area —
			// this pins bbox composition to the same semantics as `within`,
			// and fails in both directions if either filter is dropped.
			name:  "combined bbox and stop buffer partial overlap - tract",
			query: `query($bbox:BoundingBox, $stop_ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", location:{bbox:$bbox, stop_buffer:{stop_ids:$stop_ids, radius:100}}}) { name geoid geometry_area intersection_area }} }`,
			vars: hw{
				"stop_ids": []int{bartFtvlStopId},
				"bbox":     hw{"min_lon": -122.5, "min_lat": 37.5, "max_lon": -122.224175, "max_lat": 38.0},
			},
			f: func(t *testing.T, jj string) {
				testIntersectionArea(
					t,
					gjson.Get(jj, "census_datasets.0.geographies").Array(),
					1,
					1918913.384123061,
					15617.923991315816,
				)
			},
		},
		{
			// Disjoint bbox and buffer: empty clip, no geographies, rather
			// than falling back to either filter alone. The bbox is inside the
			// fixture extent and covers 22 tracts, ~10km from the FTVL stop, so
			// dropping the buffer would return those 22 rather than nothing.
			name:  "combined bbox and stop buffer disjoint - tract",
			query: `query($bbox:BoundingBox, $stop_ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", location:{bbox:$bbox, stop_buffer:{stop_ids:$stop_ids, radius:100}}}) { name geoid intersection_area }} }`,
			vars: hw{
				"stop_ids": []int{bartFtvlStopId},
				"bbox":     hw{"min_lon": -122.30, "min_lat": 37.86, "max_lon": -122.26, "max_lat": 37.88},
			},
			f: func(t *testing.T, jj string) {
				assert.Empty(t, gjson.Get(jj, "census_datasets.0.geographies").Array(), "disjoint clip must match no geographies")
			},
		},
		{
			// Radius 0 with a bbox is bbox-restricted point matching, matching
			// the `within` behavior rather than dropping the stop filter.
			name:  "combined bbox and stop buffer radius 0 - tract",
			query: `query($bbox:BoundingBox, $stop_ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", location:{bbox:$bbox, stop_buffer:{stop_ids:$stop_ids, radius:0}}}) { name geoid }} }`,
			vars: hw{
				"stop_ids": []int{bartFtvlStopId},
				"bbox":     hw{"min_lon": -123.0, "min_lat": 37.0, "max_lon": -122.0, "max_lat": 38.0},
			},
			expect: `{"census_datasets":[{"name":"tiger2024","geographies":[{"geoid":"1400000US06001406100","name":"4061"}]}]}`,
		},
		{
			// Radius 0 with the stop outside the bbox: filtered out, not
			// silently matched. Sole cover for the bbox arm of the
			// area-restricted point match.
			name:  "combined bbox and stop buffer radius 0 outside bbox - tract",
			query: `query($bbox:BoundingBox, $stop_ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", location:{bbox:$bbox, stop_buffer:{stop_ids:$stop_ids, radius:0}}}) { name geoid }} }`,
			vars: hw{
				"stop_ids": []int{bartFtvlStopId},
				"bbox":     hw{"min_lon": -122.30, "min_lat": 37.86, "max_lon": -122.26, "max_lat": 37.88},
			},
			f: func(t *testing.T, jj string) {
				assert.Empty(t, gjson.Get(jj, "census_datasets.0.geographies").Array(), "stop outside the bbox must not match")
			},
		},
		{
			// `near` yields to a stop_buffer rather than preempting it, so a
			// radius-0 buffer still reaches the area-restricted point match.
			// Without the guard on the `near` branch this returns the tracts
			// around the `near` point and drops both other filters.
			name:  "bbox and stop buffer radius 0 with near - tract",
			query: `query($bbox:BoundingBox, $near:PointRadius, $stop_ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", location:{bbox:$bbox, near:$near, stop_buffer:{stop_ids:$stop_ids, radius:0}}}) { name geoid }} }`,
			vars: hw{
				"stop_ids": []int{bartFtvlStopId},
				"bbox":     hw{"min_lon": -123.0, "min_lat": 37.0, "max_lon": -122.0, "max_lat": 38.0},
				"near":     hw{"lon": -122.270, "lat": 37.805, "radius": 1000.0},
			},
			expect: `{"census_datasets":[{"name":"tiger2024","geographies":[{"geoid":"1400000US06001406100","name":"4061"}]}]}`,
		},
		{
			// `near` alongside a positive-radius stop_buffer: the buffer wins.
			// The FTVL 100m buffer touches one tract; the `near` circle covers
			// many, so this discriminates which filter took effect.
			name:  "stop buffer with near - tract",
			query: `query($near:PointRadius, $stop_ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", location:{near:$near, stop_buffer:{stop_ids:$stop_ids, radius:100}}}) { name geoid }} }`,
			vars: hw{
				"stop_ids": []int{bartFtvlStopId},
				"near":     hw{"lon": -122.270, "lat": 37.805, "radius": 1000.0},
			},
			expect: `{"census_datasets":[{"name":"tiger2024","geographies":[{"geoid":"1400000US06001406100","name":"4061"}]}]}`,
		},
		{
			// Two separate tracts are both fully covered, and the neighbors
			// that only share a boundary get no area.
			name:  "geography ids - tract",
			query: `query($ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", location:{geography_ids:$ids}}) { geoid geometry_area intersection_area }} }`,
			vars:  hw{"ids": []int{ftvlTractId, geographyId}},
			f: func(t *testing.T, jj string) {
				covered := map[string]bool{}
				for _, v := range gjson.Get(jj, "census_datasets.0.geographies").Array() {
					geoid := v.Get("geoid").String()
					if geoid == "1400000US06001406100" || geoid == "1400000US06001403000" {
						covered[geoid] = true
						assert.InDelta(t, v.Get("geometry_area").Float(), v.Get("intersection_area").Float(), 0.01, "expected %s fully covered", geoid)
					} else {
						assert.Less(t, v.Get("intersection_area").Float(), 1.0, "expected no area for neighbor %s", geoid)
					}
				}
				assert.Len(t, covered, 2, "expected both tracts")
			},
		},
		{
			// Overlapping geographies are merged: the county and a tract
			// inside it cover the county once.
			name:  "geography ids overlapping - county",
			query: `query($ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "county", location:{geography_ids:$ids}}) { geoid geometry_area intersection_area }} }`,
			vars:  hw{"ids": []int{alamedaCountyId, ftvlTractId}},
			f: func(t *testing.T, jj string) {
				testIntersectionArea(
					t,
					gjson.Get(jj, "census_datasets.0.geographies").Array(),
					1,
					countyArea,
					countyArea,
				)
			},
		},
		{
			// The county contains the whole FTVL buffer, so the result matches
			// the stop-buffer-only case.
			name:  "combined geography ids and stop buffer - tract",
			query: `query($ids:[Int!], $stop_ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", location:{geography_ids:$ids, stop_buffer:{stop_ids:$stop_ids, radius:100}}}) { name geoid geometry_area intersection_area }} }`,
			vars: hw{
				"ids":      []int{alamedaCountyId},
				"stop_ids": []int{bartFtvlStopId},
			},
			f: func(t *testing.T, jj string) {
				testIntersectionArea(
					t,
					gjson.Get(jj, "census_datasets.0.geographies").Array(),
					1,
					1918913.384123061,
					31235.844686623343,
				)
			},
		},
		{
			// The tract holding the stop matches, and a tract away from the
			// buffer leaves nothing to search. Dropping either filter would
			// return a tract for both.
			name:  "combined geography ids and stop buffer disjoint - tract",
			query: `query($inside:[Int!], $outside:[Int!], $stop_ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) { inside: geographies(where:{layer: "tract", location:{geography_ids:$inside, stop_buffer:{stop_ids:$stop_ids, radius:100}}}) { geoid } outside: geographies(where:{layer: "tract", location:{geography_ids:$outside, stop_buffer:{stop_ids:$stop_ids, radius:100}}}) { geoid } } }`,
			vars: hw{
				"inside":   []int{ftvlTractId},
				"outside":  []int{geographyId},
				"stop_ids": []int{bartFtvlStopId},
			},
			f: func(t *testing.T, jj string) {
				assert.Equal(t, []string{"1400000US06001406100"}, astr(gjson.Get(jj, "census_datasets.0.inside.#.geoid").Array()))
				assert.Empty(t, gjson.Get(jj, "census_datasets.0.outside").Array(), "disjoint clip must match no geographies")
			},
		},
		{
			// Radius 0 matches the stop's tract only when the stop is inside
			// the named geographies.
			name:  "combined geography ids and stop buffer radius 0 - tract",
			query: `query($inside:[Int!], $outside:[Int!], $stop_ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) { inside: geographies(where:{layer: "tract", location:{geography_ids:$inside, stop_buffer:{stop_ids:$stop_ids, radius:0}}}) { geoid } outside: geographies(where:{layer: "tract", location:{geography_ids:$outside, stop_buffer:{stop_ids:$stop_ids, radius:0}}}) { geoid } } }`,
			vars: hw{
				"inside":   []int{alamedaCountyId},
				"outside":  []int{geographyId},
				"stop_ids": []int{bartFtvlStopId},
			},
			f: func(t *testing.T, jj string) {
				assert.Equal(t, []string{"1400000US06001406100"}, astr(gjson.Get(jj, "census_datasets.0.inside.#.geoid").Array()))
				assert.Empty(t, gjson.Get(jj, "census_datasets.0.outside").Array(), "stop outside the geographies must not match")
			},
		},
		{
			// An id that names no geography matches nothing, and is ignored
			// beside one that does.
			name:  "geography ids unknown - county",
			query: `query($known:[Int!], $unknown:[Int!]) { census_datasets(where:{name:"tiger2024"}) { known: geographies(where:{layer: "county", location:{geography_ids:$known}}) { geoid } unknown: geographies(where:{layer: "county", location:{geography_ids:$unknown}}) { geoid } } }`,
			vars: hw{
				"known":   []int{alamedaCountyId, -1},
				"unknown": []int{-1},
			},
			f: func(t *testing.T, jj string) {
				assert.Equal(t, []string{"0500000US06001"}, astr(gjson.Get(jj, "census_datasets.0.known.#.geoid").Array()))
				assert.Empty(t, gjson.Get(jj, "census_datasets.0.unknown").Array(), "unknown id must match no geographies")
			},
		},
		{
			// `within` takes precedence: the triangle around Alameda County
			// wins over King County.
			name:  "within and geography ids - county",
			query: `query($feature:Polygon, $ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "county", location:{within:$feature, geography_ids:$ids}}) { geoid }} }`,
			vars: hw{
				"ids": []int{kingCountyId},
				"feature": hw{"type": "Polygon", "coordinates": [][][]float64{{
					{-123.77489413290716, 38.794161309061735},
					{-122.69431950796763, 35.52679604934255},
					{-119.9104881819854, 37.991860068760204},
					{-123.77489413290716, 38.794161309061735},
				}}},
			},
			selector:     "census_datasets.0.geographies.#.geoid",
			selectExpect: []string{"0500000US06001"},
		},
		{
			name:  "agency intersection areas - county",
			query: `query { agencies(where:{agency_id:"BART"}) { agency_id census_geographies(where:{layer:"county", radius:1000.0}) { name geoid geometry_area intersection_geometry intersection_area } } }`,
			vars:  vars,
			f: func(t *testing.T, jj string) {
				testIntersectionArea(
					t,
					gjson.Get(jj, "agencies.0.census_geographies").Array(),
					1,
					countyArea,
					65341023.520996064,
				)
			},
		},
		{
			name:  "agency intersection areas - tract",
			query: `query { agencies(where:{agency_id:"BART"}) { agency_id census_geographies(where:{layer:"tract", radius:100.0}) { name geoid geometry_area intersection_geometry intersection_area } } }`,
			vars:  vars,
			f: func(t *testing.T, jj string) {
				testIntersectionArea(
					t,
					gjson.Get(jj, "agencies.0.census_geographies").Array(),
					37,
					73325244.76026103,
					687170.8023190418,
				)
			},
		},
		{
			// Same as above but with dataset pinned by name, which triggers the
			// composite (layer_id, geometry) index pushdown in the builder.
			// Results must be identical to the no-dataset case above.
			name:  "agency intersection areas - tract, dataset pinned",
			query: `query { agencies(where:{agency_id:"BART"}) { agency_id census_geographies(where:{dataset:"tiger2024", layer:"tract", radius:100.0}) { name geoid geometry_area intersection_geometry intersection_area } } }`,
			vars:  vars,
			f: func(t *testing.T, jj string) {
				testIntersectionArea(
					t,
					gjson.Get(jj, "agencies.0.census_geographies").Array(),
					37,
					73325244.76026103,
					687170.8023190418,
				)
			},
		},
		{
			// The circle is subdivided before intersecting; the tracts it
			// touches still add up to its whole area.
			name:  "subdivided search area - tract",
			query: `query($feature:Polygon) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", location:{within:$feature}}) { name geoid geometry_area intersection_area }} }`,
			vars:  hw{"feature": circle},
			f: func(t *testing.T, jj string) {
				testIntersectionArea(
					t,
					gjson.Get(jj, "census_datasets.0.geographies").Array(),
					56,
					52213095.85575217,
					circleArea,
				)
			},
		},
		{
			// The county containing the circle reports the same area as the
			// sum of its tracts.
			name:  "subdivided search area - county",
			query: `query($feature:Polygon) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "county", location:{within:$feature}}) { name geoid geometry_area intersection_area }} }`,
			vars:  hw{"feature": circle},
			f: func(t *testing.T, jj string) {
				testIntersectionArea(
					t,
					gjson.Get(jj, "census_datasets.0.geographies").Array(),
					1,
					countyArea,
					circleArea,
				)
			},
		},
		{
			// A tract searched within its own outline is fully covered, and
			// the neighbors that only share its boundary get no area.
			name:  "subdivided search area - tract within its own outline",
			query: `query($feature:Polygon) { census_datasets(where:{name:"tiger2024"}) {name geographies(where:{layer: "tract", location:{within:$feature}}) { geoid geometry_area intersection_area }} }`,
			vars:  hw{"feature": tractOutline},
			f: func(t *testing.T, jj string) {
				found := false
				for _, v := range gjson.Get(jj, "census_datasets.0.geographies").Array() {
					if v.Get("geoid").String() == "1400000US06001450701" {
						found = true
						assert.InDelta(t, v.Get("geometry_area").Float(), v.Get("intersection_area").Float(), 0.01, "expected the tract fully covered")
					} else {
						assert.Less(t, v.Get("intersection_area").Float(), 1.0, "expected no area for neighbor %s", v.Get("geoid").String())
					}
				}
				assert.True(t, found, "expected the tract itself")
			},
		},
		{
			// With no radius, an agency matches the geographies containing its
			// stops, and lists each once however many stops it contains.
			name:  "agency geographies without radius - county",
			query: `query { agencies(where:{agency_id:"BART"}) { agency_id census_geographies(where:{layer:"county"}) { geoid } } }`,
			vars:  vars,
			f: func(t *testing.T, jj string) {
				geoids := gjson.Get(jj, "agencies.0.census_geographies.#.geoid").Array()
				if assert.Len(t, geoids, 1) {
					assert.Equal(t, "0500000US06001", geoids[0].String())
				}
			},
		},
		{
			name:  "agency geographies without radius - tract",
			query: `query { agencies(where:{agency_id:"BART"}) { agency_id census_geographies(where:{layer:"tract"}) { geoid } } }`,
			vars:  vars,
			f: func(t *testing.T, jj string) {
				geoids := gjson.Get(jj, "agencies.0.census_geographies.#.geoid").Array()
				assert.NotEmpty(t, geoids)
				seen := map[string]bool{}
				for _, g := range geoids {
					assert.False(t, seen[g.String()], "expected one row for %s", g.String())
					seen[g.String()] = true
				}
			},
		},
		{
			// Sibling fields share a loader batch, and each still gets the
			// columns it selects, whichever field opens the batch.
			name:  "sibling fields with different selections - dataset, layer and source",
			query: `query($ids:[Int!]) { census_datasets(where:{name:"tiger2024"}) { whole: geographies(where:{layer:"tract", location:{geography_ids:$ids}}) { geoid geometry_area } clipped: geographies(where:{layer:"tract", location:{bbox:{min_lon:-122.30, min_lat:37.75, max_lon:-122.20, max_lat:37.85}}}) { geoid intersection_area } layers { name whole: geographies(where:{location:{geography_ids:$ids}}) { geoid geometry_area } clipped: geographies(where:{location:{bbox:{min_lon:-122.30, min_lat:37.75, max_lon:-122.20, max_lat:37.85}}}) { geoid intersection_area } } sources(where:{name:"tl_2024_06_tract.zip"}) { whole: geographies(where:{location:{geography_ids:$ids}}) { geoid geometry_area } clipped: geographies(where:{location:{bbox:{min_lon:-122.30, min_lat:37.75, max_lon:-122.20, max_lat:37.85}}}) { geoid intersection_area } } } }`,
			vars:  hw{"ids": []int{ftvlTractId}},
			f: func(t *testing.T, jj string) {
				testSelectedFields(t, jj, map[string]string{
					"census_datasets.0.whole":                           "geometry_area",
					"census_datasets.0.clipped":                         "intersection_area",
					`census_datasets.0.layers.#(name=="tract").whole`:   "geometry_area",
					`census_datasets.0.layers.#(name=="tract").clipped`: "intersection_area",
					"census_datasets.0.sources.0.whole":                 "geometry_area",
					"census_datasets.0.sources.0.clipped":               "intersection_area",
				})
			},
		},
		{
			name:  "sibling fields with different selections - stop",
			query: `query { stops(where:{feed_onestop_id:"BA", stop_id:"FTVL"}) { tract: census_geographies(where:{layer:"tract", radius:100.0}) { geoid geometry_area } county: census_geographies(where:{layer:"county", radius:100.0}) { geoid intersection_area } } }`,
			vars:  vars,
			f: func(t *testing.T, jj string) {
				testSelectedFields(t, jj, map[string]string{
					"stops.0.tract":  "geometry_area",
					"stops.0.county": "intersection_area",
				})
			},
		},
	}
	queryTestcases(t, c, testcases)
}

func TestCensusResolver_DatasetValuesRelay(t *testing.T) {
	c, _ := newTestClient(t)
	vars := hw{}
	testcases := []testcase{
		// Census dataset values tests
		{
			name:              "census_datasets values_relay basic query",
			query:             `query { census_datasets(where:{name:"acsdt5y2022"}) { values_relay(first:10) { edges { node { geoid dataset_name source_name values } } pageInfo { hasNextPage endCursor } } } }`,
			vars:              vars,
			selectExpectCount: 10,
			selector:          "census_datasets.0.values_relay.edges.#.node.geoid",
		},
		{
			name:              "census_datasets values_relay filter by table",
			query:             `query { census_datasets(where:{name:"acsdt5y2022"}) { values_relay(first:50, where:{table:"b01001"}) { edges { node { geoid dataset_name values table { table_name } } } pageInfo { hasNextPage } } } }`,
			vars:              vars,
			selector:          "census_datasets.0.values_relay.edges.#.node.table.table_name",
			selectExpectCount: 50,
			f: func(t *testing.T, jj string) {
				tables := gjson.Get(jj, "census_datasets.0.values_relay.edges.#.node.table.table_name").Array()
				for _, tbl := range tables {
					assert.Equal(t, "b01001", tbl.String(), "all results should be from b01001 table")
				}
			},
		},
		{
			name:   "census_datasets values_relay filter by exact geoid - county",
			query:  `query { census_datasets(where:{name:"acsdt5y2022"}) { values_relay(where:{table:"b01001", geoid:"0500000US06001"}) { edges { node { geoid dataset_name source_name values } } pageInfo { hasNextPage } } } }`,
			vars:   vars,
			expect: `{"census_datasets":[{"values_relay":{"edges":[{"node":{"dataset_name":"acsdt5y2022","geoid":"0500000US06001","source_name":"acsdt5y2022-b01001.dat","values":{"b01001_001":1663823,"b01001_002":826561,"b01001_003":45925,"b01001_004":47576,"b01001_005":48382,"b01001_006":28544,"b01001_007":20330,"b01001_008":10059,"b01001_009":9868,"b01001_010":28353,"b01001_011":62380,"b01001_012":72338,"b01001_013":69473,"b01001_014":62112,"b01001_015":57592,"b01001_016":55360,"b01001_017":52792,"b01001_018":20592,"b01001_019":27469,"b01001_020":16861,"b01001_021":19794,"b01001_022":30165,"b01001_023":17775,"b01001_024":12014,"b01001_025":10807,"b01001_026":837262,"b01001_027":44062,"b01001_028":44832,"b01001_029":46225,"b01001_030":27281,"b01001_031":20424,"b01001_032":9802,"b01001_033":10241,"b01001_034":28031,"b01001_035":61402,"b01001_036":70642,"b01001_037":67277,"b01001_038":59739,"b01001_039":55980,"b01001_040":53250,"b01001_041":53257,"b01001_042":21514,"b01001_043":27480,"b01001_044":17165,"b01001_045":23991,"b01001_046":35310,"b01001_047":23707,"b01001_048":15582,"b01001_049":20068}}}],"pageInfo":{"hasNextPage":false}}}]}`,
		},
		{
			name:   "census_datasets values_relay filter by exact geoid - tract",
			query:  `query { census_datasets(where:{name:"acsdt5y2022"}) { values_relay(where:{table:"b01001", geoid:"1400000US06001403000"}) { edges { node { geoid dataset_name source_name values } } pageInfo { hasNextPage } } } }`,
			vars:   vars,
			expect: `{"census_datasets":[{"values_relay":{"edges":[{"node":{"dataset_name":"acsdt5y2022","geoid":"1400000US06001403000","source_name":"acsdt5y2022-b01001.dat","values":{"b01001_001":2897,"b01001_002":1241,"b01001_003":8,"b01001_004":0,"b01001_005":17,"b01001_006":22,"b01001_007":3,"b01001_008":0,"b01001_009":17,"b01001_010":25,"b01001_011":108,"b01001_012":142,"b01001_013":89,"b01001_014":154,"b01001_015":87,"b01001_016":39,"b01001_017":53,"b01001_018":14,"b01001_019":77,"b01001_020":22,"b01001_021":30,"b01001_022":89,"b01001_023":81,"b01001_024":37,"b01001_025":127,"b01001_026":1656,"b01001_027":45,"b01001_028":65,"b01001_029":22,"b01001_030":13,"b01001_031":32,"b01001_032":0,"b01001_033":4,"b01001_034":13,"b01001_035":146,"b01001_036":208,"b01001_037":57,"b01001_038":66,"b01001_039":37,"b01001_040":130,"b01001_041":67,"b01001_042":34,"b01001_043":10,"b01001_044":110,"b01001_045":44,"b01001_046":68,"b01001_047":159,"b01001_048":164,"b01001_049":162}}}],"pageInfo":{"hasNextPage":false}}}]}`,
		},
		{
			name:         "census_datasets values_relay filter by geoid_prefix - tract prefix",
			query:        `query { census_datasets(where:{name:"acsdt5y2022"}) { values_relay(first:20, where:{table:"b01001", geoid_prefix:"1400000US0600140"}) { edges { node { geoid dataset_name values } } pageInfo { hasNextPage endCursor } } } }`,
			vars:         vars,
			selector:     "census_datasets.0.values_relay.edges.#.node.geoid",
			selectExpect: []string{"1400000US06001400100", "1400000US06001400200", "1400000US06001400300", "1400000US06001400400", "1400000US06001400500", "1400000US06001400600", "1400000US06001400700", "1400000US06001400800", "1400000US06001400900", "1400000US06001401000", "1400000US06001401100", "1400000US06001401200", "1400000US06001401300", "1400000US06001401400", "1400000US06001401500", "1400000US06001401600", "1400000US06001401700", "1400000US06001401800", "1400000US06001402200", "1400000US06001402400"},
		},
		{
			name:              "census_datasets values_relay filter by geoid_prefix - county prefix",
			query:             `query { census_datasets(where:{name:"acsdt5y2022"}) { values_relay(first:10, where:{table:"b01001", geoid_prefix:"0500000US"}) { edges { node { geoid dataset_name values } } pageInfo { hasNextPage } } } }`,
			vars:              vars,
			selector:          "census_datasets.0.values_relay.edges.#.node.geoid",
			selectExpectCount: 2,
			selectExpect:      []string{"0500000US06001", "0500000US53033"},
		},
		{
			name:   "census_datasets values_relay combined filters - table",
			query:  `query { census_datasets(where:{name:"acsdt5y2022"}) { values_relay(first:5, where:{table:"b01001"}) { edges { node { geoid dataset_name values table { table_name } } } pageInfo { hasNextPage endCursor } } } }`,
			vars:   vars,
			expect: `{"census_datasets":[{"values_relay":{"edges":[{"node":{"dataset_name":"acsdt5y2022","geoid":"0500000US06001","table":{"table_name":"b01001"},"values":{"b01001_001":1663823,"b01001_002":826561,"b01001_003":45925,"b01001_004":47576,"b01001_005":48382,"b01001_006":28544,"b01001_007":20330,"b01001_008":10059,"b01001_009":9868,"b01001_010":28353,"b01001_011":62380,"b01001_012":72338,"b01001_013":69473,"b01001_014":62112,"b01001_015":57592,"b01001_016":55360,"b01001_017":52792,"b01001_018":20592,"b01001_019":27469,"b01001_020":16861,"b01001_021":19794,"b01001_022":30165,"b01001_023":17775,"b01001_024":12014,"b01001_025":10807,"b01001_026":837262,"b01001_027":44062,"b01001_028":44832,"b01001_029":46225,"b01001_030":27281,"b01001_031":20424,"b01001_032":9802,"b01001_033":10241,"b01001_034":28031,"b01001_035":61402,"b01001_036":70642,"b01001_037":67277,"b01001_038":59739,"b01001_039":55980,"b01001_040":53250,"b01001_041":53257,"b01001_042":21514,"b01001_043":27480,"b01001_044":17165,"b01001_045":23991,"b01001_046":35310,"b01001_047":23707,"b01001_048":15582,"b01001_049":20068}}},{"node":{"dataset_name":"acsdt5y2022","geoid":"0500000US53033","table":{"table_name":"b01001"},"values":{"b01001_001":2254371,"b01001_002":1143593,"b01001_003":62532,"b01001_004":64195,"b01001_005":63784,"b01001_006":37244,"b01001_007":25041,"b01001_008":12383,"b01001_009":11862,"b01001_010":45530,"b01001_011":104190,"b01001_012":111644,"b01001_013":98996,"b01001_014":84701,"b01001_015":77536,"b01001_016":74586,"b01001_017":68869,"b01001_018":26439,"b01001_019":36908,"b01001_020":21287,"b01001_021":29257,"b01001_022":36389,"b01001_023":22150,"b01001_024":13430,"b01001_025":14640,"b01001_026":1110778,"b01001_027":59182,"b01001_028":59675,"b01001_029":62983,"b01001_030":35726,"b01001_031":25271,"b01001_032":11745,"b01001_033":11781,"b01001_034":42801,"b01001_035":94191,"b01001_036":99489,"b01001_037":88009,"b01001_038":79800,"b01001_039":72650,"b01001_040":69551,"b01001_041":66704,"b01001_042":24989,"b01001_043":37329,"b01001_044":21832,"b01001_045":30582,"b01001_046":44213,"b01001_047":27777,"b01001_048":18830,"b01001_049":25668}}},{"node":{"dataset_name":"acsdt5y2022","geoid":"1400000US06001400100","table":{"table_name":"b01001"},"values":{"b01001_001":3269,"b01001_002":1621,"b01001_003":72,"b01001_004":105,"b01001_005":167,"b01001_006":28,"b01001_007":0,"b01001_008":0,"b01001_009":0,"b01001_010":31,"b01001_011":40,"b01001_012":116,"b01001_013":30,"b01001_014":70,"b01001_015":85,"b01001_016":201,"b01001_017":185,"b01001_018":91,"b01001_019":37,"b01001_020":19,"b01001_021":76,"b01001_022":114,"b01001_023":56,"b01001_024":28,"b01001_025":70,"b01001_026":1648,"b01001_027":61,"b01001_028":135,"b01001_029":59,"b01001_030":34,"b01001_031":10,"b01001_032":0,"b01001_033":24,"b01001_034":21,"b01001_035":43,"b01001_036":69,"b01001_037":146,"b01001_038":83,"b01001_039":88,"b01001_040":184,"b01001_041":118,"b01001_042":0,"b01001_043":52,"b01001_044":56,"b01001_045":98,"b01001_046":88,"b01001_047":74,"b01001_048":92,"b01001_049":113}}},{"node":{"dataset_name":"acsdt5y2022","geoid":"1400000US06001400200","table":{"table_name":"b01001"},"values":{"b01001_001":2147,"b01001_002":1075,"b01001_003":83,"b01001_004":35,"b01001_005":21,"b01001_006":20,"b01001_007":10,"b01001_008":4,"b01001_009":0,"b01001_010":12,"b01001_011":111,"b01001_012":108,"b01001_013":120,"b01001_014":54,"b01001_015":69,"b01001_016":70,"b01001_017":54,"b01001_018":10,"b01001_019":51,"b01001_020":19,"b01001_021":35,"b01001_022":95,"b01001_023":50,"b01001_024":21,"b01001_025":23,"b01001_026":1072,"b01001_027":86,"b01001_028":7,"b01001_029":59,"b01001_030":39,"b01001_031":11,"b01001_032":0,"b01001_033":0,"b01001_034":20,"b01001_035":108,"b01001_036":64,"b01001_037":118,"b01001_038":27,"b01001_039":87,"b01001_040":41,"b01001_041":42,"b01001_042":13,"b01001_043":40,"b01001_044":16,"b01001_045":35,"b01001_046":85,"b01001_047":116,"b01001_048":47,"b01001_049":11}}},{"node":{"dataset_name":"acsdt5y2022","geoid":"1400000US06001400300","table":{"table_name":"b01001"},"values":{"b01001_001":5619,"b01001_002":2801,"b01001_003":34,"b01001_004":206,"b01001_005":161,"b01001_006":149,"b01001_007":69,"b01001_008":0,"b01001_009":6,"b01001_010":27,"b01001_011":386,"b01001_012":348,"b01001_013":253,"b01001_014":154,"b01001_015":231,"b01001_016":171,"b01001_017":180,"b01001_018":46,"b01001_019":67,"b01001_020":0,"b01001_021":31,"b01001_022":120,"b01001_023":103,"b01001_024":28,"b01001_025":31,"b01001_026":2818,"b01001_027":93,"b01001_028":194,"b01001_029":75,"b01001_030":30,"b01001_031":30,"b01001_032":8,"b01001_033":23,"b01001_034":0,"b01001_035":400,"b01001_036":288,"b01001_037":192,"b01001_038":162,"b01001_039":248,"b01001_040":322,"b01001_041":46,"b01001_042":16,"b01001_043":88,"b01001_044":73,"b01001_045":65,"b01001_046":186,"b01001_047":107,"b01001_048":18,"b01001_049":154}}}],"pageInfo":{"endCursor":"MTQwMDAwMFVTMDYwMDE0MDAzMDAsOQ","hasNextPage":true}}}]}`,
		},
		{
			name:   "census_datasets values_relay combined filters - geoid",
			query:  `query { census_datasets(where:{name:"acsdt5y2022"}) { values_relay(where:{table:"b01001", geoid:"0500000US06001"}) { edges { node { geoid dataset_name source_name values } } pageInfo { hasNextPage } } } }`,
			vars:   vars,
			expect: `{"census_datasets":[{"values_relay":{"edges":[{"node":{"dataset_name":"acsdt5y2022","geoid":"0500000US06001","source_name":"acsdt5y2022-b01001.dat","values":{"b01001_001":1663823,"b01001_002":826561,"b01001_003":45925,"b01001_004":47576,"b01001_005":48382,"b01001_006":28544,"b01001_007":20330,"b01001_008":10059,"b01001_009":9868,"b01001_010":28353,"b01001_011":62380,"b01001_012":72338,"b01001_013":69473,"b01001_014":62112,"b01001_015":57592,"b01001_016":55360,"b01001_017":52792,"b01001_018":20592,"b01001_019":27469,"b01001_020":16861,"b01001_021":19794,"b01001_022":30165,"b01001_023":17775,"b01001_024":12014,"b01001_025":10807,"b01001_026":837262,"b01001_027":44062,"b01001_028":44832,"b01001_029":46225,"b01001_030":27281,"b01001_031":20424,"b01001_032":9802,"b01001_033":10241,"b01001_034":28031,"b01001_035":61402,"b01001_036":70642,"b01001_037":67277,"b01001_038":59739,"b01001_039":55980,"b01001_040":53250,"b01001_041":53257,"b01001_042":21514,"b01001_043":27480,"b01001_044":17165,"b01001_045":23991,"b01001_046":35310,"b01001_047":23707,"b01001_048":15582,"b01001_049":20068}}}],"pageInfo":{"hasNextPage":false}}}]}`,
		},
		{
			name:              "census_datasets values_relay combined filters - table and geoid_prefix",
			query:             `query { census_datasets(where:{name:"acsdt5y2022"}) { values_relay(first:10, where:{table:"b01001", geoid_prefix:"1400000US0600140"}) { edges { node { geoid dataset_name values table { table_name } } } pageInfo { hasNextPage endCursor } } } }`,
			vars:              vars,
			selector:          "census_datasets.0.values_relay.edges.#.node.geoid",
			selectExpectCount: 10,
			f: func(t *testing.T, jj string) {
				tables := gjson.Get(jj, "census_datasets.0.values_relay.edges.#.node.table.table_name").Array()
				geoids := gjson.Get(jj, "census_datasets.0.values_relay.edges.#.node.geoid").Array()
				for _, tbl := range tables {
					assert.Equal(t, "b01001", tbl.String(), "all results should be from b01001 table")
				}
				for _, geoid := range geoids {
					assert.Contains(t, geoid.String(), "1400000US0600140", "all geoids should start with the prefix")
				}
			},
		},
		{
			name:   "census_datasets values_relay combined filters - all filters",
			query:  `query { census_datasets(where:{name:"acsdt5y2022"}) { values_relay(where:{table:"b01001", geoid:"1400000US06001403000"}) { edges { node { geoid dataset_name source_name values table { table_name } } } pageInfo { hasNextPage } } } }`,
			vars:   vars,
			expect: `{"census_datasets":[{"values_relay":{"edges":[{"node":{"dataset_name":"acsdt5y2022","geoid":"1400000US06001403000","source_name":"acsdt5y2022-b01001.dat","table":{"table_name":"b01001"},"values":{"b01001_001":2897,"b01001_002":1241,"b01001_003":8,"b01001_004":0,"b01001_005":17,"b01001_006":22,"b01001_007":3,"b01001_008":0,"b01001_009":17,"b01001_010":25,"b01001_011":108,"b01001_012":142,"b01001_013":89,"b01001_014":154,"b01001_015":87,"b01001_016":39,"b01001_017":53,"b01001_018":14,"b01001_019":77,"b01001_020":22,"b01001_021":30,"b01001_022":89,"b01001_023":81,"b01001_024":37,"b01001_025":127,"b01001_026":1656,"b01001_027":45,"b01001_028":65,"b01001_029":22,"b01001_030":13,"b01001_031":32,"b01001_032":0,"b01001_033":4,"b01001_034":13,"b01001_035":146,"b01001_036":208,"b01001_037":57,"b01001_038":66,"b01001_039":37,"b01001_040":130,"b01001_041":67,"b01001_042":34,"b01001_043":10,"b01001_044":110,"b01001_045":44,"b01001_046":68,"b01001_047":159,"b01001_048":164,"b01001_049":162}}}],"pageInfo":{"hasNextPage":false}}}]}`,
		},
		{
			name:   "census_datasets values_relay no results with non-existent geoid",
			query:  `query { census_datasets(where:{name:"acsdt5y2022"}) { values_relay(where:{geoid:"9999999999999"}) { edges { node { geoid dataset_name values } } pageInfo { hasNextPage } } } }`,
			vars:   vars,
			expect: `{"census_datasets":[{"values_relay":{"edges":[],"pageInfo":{"hasNextPage":false}}}]}`,
		},
		{
			name:   "census_datasets values_relay no results with non-existent table",
			query:  `query { census_datasets(where:{name:"acsdt5y2022"}) { values_relay(where:{table:"nonexistent"}) { edges { node { geoid dataset_name values } } pageInfo { hasNextPage } } } }`,
			vars:   vars,
			expect: `{"census_datasets":[{"values_relay":{"edges":[],"pageInfo":{"hasNextPage":false}}}]}`,
		},
		{
			name:  "census_datasets values_relay pagination test",
			query: `query { census_datasets(where:{name:"acsdt5y2022"}) { values_relay(first:5, where:{geoid_prefix:"1400000US0600140"}) { edges { node { geoid } cursor } pageInfo { hasNextPage endCursor } } } }`,
			vars:  vars,
			f: func(t *testing.T, jj string) {
				hasNextPage := gjson.Get(jj, "census_datasets.0.values_relay.pageInfo.hasNextPage").Bool()
				endCursor := gjson.Get(jj, "census_datasets.0.values_relay.pageInfo.endCursor").String()
				assert.True(t, hasNextPage, "should have next page")
				assert.NotEmpty(t, endCursor, "should have end cursor")
				edges := gjson.Get(jj, "census_datasets.0.values_relay.edges").Array()
				assert.Equal(t, 5, len(edges), "should return 5 results")
			},
		},
	}
	queryTestcases(t, c, testcases)
}

func TestCensusResolver_Antimeridian(t *testing.T) {
	c, _ := newTestClient(t)
	q := `query($bbox:BoundingBox, $within:Polygon, $near:PointRadius) { census_datasets(where:{name:"tiger2024"}) { geographies(where:{layer:"tract", location:{bbox:$bbox, within:$within, near:$near}}) { geoid intersection_area } } }`
	testcases := []struct {
		name string
		vars hw
	}{
		{"bbox past 180", hw{"bbox": hw{"min_lon": 178.0, "min_lat": 51.0, "max_lon": 182.0, "max_lat": 53.0}}},
		{"bbox with min_lon east of max_lon", hw{"bbox": hw{"min_lon": 170.0, "min_lat": 51.0, "max_lon": -170.0, "max_lat": 53.0}}},
		{"within drawn across 180", hw{"within": hw{"type": "Polygon", "coordinates": [][][]float64{{{179, 51}, {-179, 51}, {-179, 53}, {179, 53}, {179, 51}}}}}},
		{"near circle across 180", hw{"near": hw{"lon": 179.99, "lat": 51.9, "radius": 5000.0}}},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var resp map[string]interface{}
			opts := []client.Option{}
			for k, v := range tc.vars {
				opts = append(opts, client.Var(k, v))
			}
			err := c.Post(q, &resp, opts...)
			assert.ErrorContains(t, err, "antimeridian")
		})
	}
}

func testIntersectionArea(t *testing.T, a []gjson.Result, expectCount int, expectGeometryArea float64, expectIntersectionArea float64) {
	geometryAreas := map[string]float64{}
	intersectionArea := 0.0
	for _, v := range a {
		intersectionArea += v.Get("intersection_area").Float()
		geometryAreas[v.Get("geoid").String()] = v.Get("geometry_area").Float()
	}
	geometryArea := 0.0
	for _, v := range geometryAreas {
		geometryArea += v
	}
	assert.InDelta(t, expectIntersectionArea, intersectionArea, 1.0, "expected intersection area")
	assert.InDelta(t, expectGeometryArea, geometryArea, 1.0, "expected geometry area")
	assert.Equal(t, expectCount, len(a), "expected geographies returned")
	assert.Equal(t, len(geometryAreas), len(a), "expected one row per geography")
}

// testSelectedFields checks that every row at each path has a value for the
// field selected there.
func testSelectedFields(t *testing.T, jj string, fields map[string]string) {
	for path, field := range fields {
		rows := gjson.Get(jj, path).Array()
		assert.NotEmpty(t, rows, "expected rows at %s", path)
		nulls := 0
		for _, v := range rows {
			if v.Get(field).Type == gjson.Null {
				nulls++
			}
		}
		assert.Zero(t, nulls, "expected %s on every row at %s", field, path)
	}
}
