package gql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestCensusResolver(t *testing.T) {
	c, cfg := newTestClient(t)
	// Define checks and get IDs for tests
	countyArea := 2126920288.43 // Area of Alameda County in m^2
	geographyId := 0
	if err := cfg.Finder.DBX().QueryRowx(`select id from tl_census_geographies where geoid = '1400000US06001403000'`).Scan(&geographyId); err != nil {
		t.Errorf("could not get geography id for test: %s", err.Error())
	}
	bartFtvlStopId := 0
	if err := cfg.Finder.DBX().QueryRowx(`select gtfs_stops.id from gtfs_stops join feed_states using(feed_version_id) where stop_id = 'FTVL'`).Scan(&bartFtvlStopId); err != nil {
		t.Errorf("could not get stop id for test: %s", err.Error())
	}
	bartMcarStopId := 0
	if err := cfg.Finder.DBX().QueryRowx(`select gtfs_stops.id from gtfs_stops join feed_states using(feed_version_id) where stop_id = 'MCAR'`).Scan(&bartMcarStopId); err != nil {
		t.Errorf("could not get stop id for test: %s", err.Error())
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
					1918910.47033,
					31235.844716912135,
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
					9.496240598676275e+06,
					3.123584448019881e+06,
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
					3.9412406339011015e+06,
					62472.080208225176,
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
					4755614.60179,
					829385.7985148486,
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
					829385.7985148486,
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
			name:  "agency intersection areas - county",
			query: `query { agencies(where:{agency_id:"BART"}) { agency_id census_geographies(where:{layer:"county", radius:1000.0}) { name geoid geometry_area intersection_geometry intersection_area } } }`,
			vars:  vars,
			f: func(t *testing.T, jj string) {
				testIntersectionArea(
					t,
					gjson.Get(jj, "agencies.0.census_geographies").Array(),
					18,
					countyArea,
					65341022.43,
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
					39,
					73325034.5592,
					687170.8023156085,
				)
			},
		},
		// Census values tests
		{
			name:              "census_values basic query",
			query:             `query { census_values(first:10) { edges { node { geoid dataset_name source_name values } } pageInfo { hasNextPage endCursor } } }`,
			vars:              vars,
			selectExpectCount: 10,
			selector:          "census_values.edges.#.node.geoid",
		},
		{
			name:              "census_values filter by dataset",
			query:             `query { census_values(first:100, where:{dataset:"acsdt5y2022"}) { edges { node { geoid dataset_name source_name values } } pageInfo { hasNextPage } } }`,
			vars:              vars,
			selector:          "census_values.edges.#.node.dataset_name",
			selectExpectCount: 100,
			f: func(t *testing.T, jj string) {
				datasets := gjson.Get(jj, "census_values.edges.#.node.dataset_name").Array()
				for _, ds := range datasets {
					assert.Equal(t, "acsdt5y2022", ds.String(), "all results should be from acsdt5y2022 dataset")
				}
			},
		},
		{
			name:              "census_values filter by table",
			query:             `query { census_values(first:50, where:{table:"b01001"}) { edges { node { geoid dataset_name values table { table_name } } } pageInfo { hasNextPage } } }`,
			vars:              vars,
			selector:          "census_values.edges.#.node.table.table_name",
			selectExpectCount: 50,
			f: func(t *testing.T, jj string) {
				tables := gjson.Get(jj, "census_values.edges.#.node.table.table_name").Array()
				for _, tbl := range tables {
					assert.Equal(t, "b01001", tbl.String(), "all results should be from b01001 table")
				}
			},
		},
		{
			name:   "census_values filter by exact geoid - county",
			query:  `query { census_values(where:{geoid:"0500000US06001"}) { edges { node { geoid dataset_name source_name values } } pageInfo { hasNextPage } } }`,
			vars:   vars,
			expect: `{"census_values":{"edges":[{"node":{"dataset_name":"acsdt5y2022","geoid":"0500000US06001","source_name":"acsdt5y2022-b01001.dat","values":{"b01001_001":1663823,"b01001_002":826561,"b01001_003":45925,"b01001_004":47576,"b01001_005":48382,"b01001_006":28544,"b01001_007":20330,"b01001_008":10059,"b01001_009":9868,"b01001_010":28353,"b01001_011":62380,"b01001_012":72338,"b01001_013":69473,"b01001_014":62112,"b01001_015":57592,"b01001_016":55360,"b01001_017":52792,"b01001_018":20592,"b01001_019":27469,"b01001_020":16861,"b01001_021":19794,"b01001_022":30165,"b01001_023":17775,"b01001_024":12014,"b01001_025":10807,"b01001_026":837262,"b01001_027":44062,"b01001_028":44832,"b01001_029":46225,"b01001_030":27281,"b01001_031":20424,"b01001_032":9802,"b01001_033":10241,"b01001_034":28031,"b01001_035":61402,"b01001_036":70642,"b01001_037":67277,"b01001_038":59739,"b01001_039":55980,"b01001_040":53250,"b01001_041":53257,"b01001_042":21514,"b01001_043":27480,"b01001_044":17165,"b01001_045":23991,"b01001_046":35310,"b01001_047":23707,"b01001_048":15582,"b01001_049":20068}}}],"pageInfo":{"hasNextPage":false}}}`,
		},
		{
			name:   "census_values filter by exact geoid - tract",
			query:  `query { census_values(where:{geoid:"1400000US06001403000"}) { edges { node { geoid dataset_name source_name values } } pageInfo { hasNextPage } } }`,
			vars:   vars,
			expect: `{"census_values":{"edges":[{"node":{"dataset_name":"acsdt5y2022","geoid":"1400000US06001403000","source_name":"acsdt5y2022-b01001.dat","values":{"b01001_001":4758,"b01001_002":1875,"b01001_003":75,"b01001_004":37,"b01001_005":100,"b01001_006":8,"b01001_007":40,"b01001_008":0,"b01001_009":0,"b01001_010":133,"b01001_011":275,"b01001_012":113,"b01001_013":325,"b01001_014":174,"b01001_015":162,"b01001_016":96,"b01001_017":94,"b01001_018":35,"b01001_019":27,"b01001_020":29,"b01001_021":31,"b01001_022":93,"b01001_023":21,"b01001_024":0,"b01001_025":7,"b01001_026":2883,"b01001_027":216,"b01001_028":123,"b01001_029":203,"b01001_030":74,"b01001_031":34,"b01001_032":0,"b01001_033":17,"b01001_034":6,"b01001_035":429,"b01001_036":258,"b01001_037":320,"b01001_038":137,"b01001_039":243,"b01001_040":167,"b01001_041":222,"b01001_042":32,"b01001_043":108,"b01001_044":10,"b01001_045":60,"b01001_046":45,"b01001_047":91,"b01001_048":20,"b01001_049":68}}}],"pageInfo":{"hasNextPage":false}}}`,
		},
		{
			name:         "census_values filter by geoid_prefix - tract prefix",
			query:        `query { census_values(first:20, where:{geoid_prefix:"1400000US0600140"}) { edges { node { geoid dataset_name values } } pageInfo { hasNextPage endCursor } } }`,
			vars:         vars,
			selector:     "census_values.edges.#.node.geoid",
			selectExpect: []string{"1400000US06001400100", "1400000US06001400200", "1400000US06001400300", "1400000US06001400400", "1400000US06001400500", "1400000US06001400600", "1400000US06001400700", "1400000US06001400800", "1400000US06001400900", "1400000US06001401000", "1400000US06001401100", "1400000US06001401200", "1400000US06001401300", "1400000US06001401400", "1400000US06001401500", "1400000US06001401600"},
		},
		{
			name:              "census_values filter by geoid_prefix - county prefix",
			query:             `query { census_values(first:10, where:{geoid_prefix:"0500000US"}) { edges { node { geoid dataset_name values } } pageInfo { hasNextPage } } }`,
			vars:              vars,
			selector:          "census_values.edges.#.node.geoid",
			selectExpectCount: 2,
			selectExpect:      []string{"0500000US06001", "0500000US53033"},
		},
		{
			name:   "census_values combined filters - dataset and table",
			query:  `query { census_values(first:5, where:{dataset:"acsdt5y2022", table:"b01001"}) { edges { node { geoid dataset_name values table { table_name } } } pageInfo { hasNextPage endCursor } } }`,
			vars:   vars,
			expect: `{"census_values":{"edges":[{"node":{"dataset_name":"acsdt5y2022","geoid":"0500000US06001","table":{"table_name":"b01001"},"values":{"b01001_001":1663823,"b01001_002":826561,"b01001_003":45925,"b01001_004":47576,"b01001_005":48382,"b01001_006":28544,"b01001_007":20330,"b01001_008":10059,"b01001_009":9868,"b01001_010":28353,"b01001_011":62380,"b01001_012":72338,"b01001_013":69473,"b01001_014":62112,"b01001_015":57592,"b01001_016":55360,"b01001_017":52792,"b01001_018":20592,"b01001_019":27469,"b01001_020":16861,"b01001_021":19794,"b01001_022":30165,"b01001_023":17775,"b01001_024":12014,"b01001_025":10807,"b01001_026":837262,"b01001_027":44062,"b01001_028":44832,"b01001_029":46225,"b01001_030":27281,"b01001_031":20424,"b01001_032":9802,"b01001_033":10241,"b01001_034":28031,"b01001_035":61402,"b01001_036":70642,"b01001_037":67277,"b01001_038":59739,"b01001_039":55980,"b01001_040":53250,"b01001_041":53257,"b01001_042":21514,"b01001_043":27480,"b01001_044":17165,"b01001_045":23991,"b01001_046":35310,"b01001_047":23707,"b01001_048":15582,"b01001_049":20068}}},{"node":{"dataset_name":"acsdt5y2022","geoid":"0500000US53033","table":{"table_name":"b01001"},"values":{"b01001_001":2254371,"b01001_002":1143593,"b01001_003":62532,"b01001_004":64195,"b01001_005":63784,"b01001_006":37244,"b01001_007":25041,"b01001_008":12383,"b01001_009":11862,"b01001_010":45530,"b01001_011":104190,"b01001_012":111644,"b01001_013":98996,"b01001_014":84701,"b01001_015":77536,"b01001_016":74586,"b01001_017":68869,"b01001_018":26439,"b01001_019":36908,"b01001_020":21287,"b01001_021":29257,"b01001_022":36389,"b01001_023":22150,"b01001_024":13430,"b01001_025":14640,"b01001_026":1110778,"b01001_027":59182,"b01001_028":59675,"b01001_029":62983,"b01001_030":35726,"b01001_031":25271,"b01001_032":11745,"b01001_033":11781,"b01001_034":42801,"b01001_035":94191,"b01001_036":99489,"b01001_037":88009,"b01001_038":79800,"b01001_039":72650,"b01001_040":69551,"b01001_041":66704,"b01001_042":24989,"b01001_043":37329,"b01001_044":21832,"b01001_045":30582,"b01001_046":44213,"b01001_047":27777,"b01001_048":18830,"b01001_049":25668}}},{"node":{"dataset_name":"acsdt5y2022","geoid":"1400000US06001400100","table":{"table_name":"b01001"},"values":{"b01001_001":3269,"b01001_002":1621,"b01001_003":72,"b01001_004":105,"b01001_005":167,"b01001_006":28,"b01001_007":0,"b01001_008":0,"b01001_009":0,"b01001_010":31,"b01001_011":40,"b01001_012":116,"b01001_013":30,"b01001_014":70,"b01001_015":85,"b01001_016":201,"b01001_017":185,"b01001_018":91,"b01001_019":37,"b01001_020":19,"b01001_021":76,"b01001_022":114,"b01001_023":56,"b01001_024":28,"b01001_025":70,"b01001_026":1648,"b01001_027":61,"b01001_028":135,"b01001_029":59,"b01001_030":34,"b01001_031":10,"b01001_032":0,"b01001_033":24,"b01001_034":21,"b01001_035":43,"b01001_036":69,"b01001_037":146,"b01001_038":83,"b01001_039":88,"b01001_040":184,"b01001_041":118,"b01001_042":0,"b01001_043":52,"b01001_044":56,"b01001_045":98,"b01001_046":88,"b01001_047":74,"b01001_048":92,"b01001_049":113}}},{"node":{"dataset_name":"acsdt5y2022","geoid":"1400000US06001400200","table":{"table_name":"b01001"},"values":{"b01001_001":2147,"b01001_002":1075,"b01001_003":83,"b01001_004":35,"b01001_005":21,"b01001_006":20,"b01001_007":10,"b01001_008":4,"b01001_009":0,"b01001_010":12,"b01001_011":111,"b01001_012":108,"b01001_013":120,"b01001_014":54,"b01001_015":69,"b01001_016":70,"b01001_017":54,"b01001_018":10,"b01001_019":51,"b01001_020":19,"b01001_021":35,"b01001_022":95,"b01001_023":50,"b01001_024":21,"b01001_025":23,"b01001_026":1072,"b01001_027":86,"b01001_028":7,"b01001_029":59,"b01001_030":39,"b01001_031":11,"b01001_032":0,"b01001_033":0,"b01001_034":20,"b01001_035":108,"b01001_036":64,"b01001_037":118,"b01001_038":27,"b01001_039":87,"b01001_040":41,"b01001_041":42,"b01001_042":13,"b01001_043":40,"b01001_044":16,"b01001_045":35,"b01001_046":85,"b01001_047":116,"b01001_048":47,"b01001_049":11}}},{"node":{"dataset_name":"acsdt5y2022","geoid":"1400000US06001400300","table":{"table_name":"b01001"},"values":{"b01001_001":5619,"b01001_002":2801,"b01001_003":34,"b01001_004":206,"b01001_005":161,"b01001_006":149,"b01001_007":69,"b01001_008":0,"b01001_009":6,"b01001_010":27,"b01001_011":386,"b01001_012":348,"b01001_013":253,"b01001_014":154,"b01001_015":231,"b01001_016":171,"b01001_017":180,"b01001_018":46,"b01001_019":67,"b01001_020":0,"b01001_021":31,"b01001_022":120,"b01001_023":103,"b01001_024":28,"b01001_025":31,"b01001_026":2818,"b01001_027":93,"b01001_028":194,"b01001_029":75,"b01001_030":30,"b01001_031":30,"b01001_032":8,"b01001_033":23,"b01001_034":0,"b01001_035":400,"b01001_036":288,"b01001_037":192,"b01001_038":162,"b01001_039":248,"b01001_040":322,"b01001_041":46,"b01001_042":16,"b01001_043":88,"b01001_044":73,"b01001_045":65,"b01001_046":186,"b01001_047":107,"b01001_048":18,"b01001_049":154}}}],"pageInfo":{"endCursor":"MTQwMDAwMFVTMDYwMDE0MDAzMDAsOQ","hasNextPage":true}}}`,
		},
		{
			name:   "census_values combined filters - dataset and geoid",
			query:  `query { census_values(where:{dataset:"acsdt5y2022", geoid:"0500000US06001"}) { edges { node { geoid dataset_name source_name values } } pageInfo { hasNextPage } } }`,
			vars:   vars,
			expect: `{"census_values":{"edges":[{"node":{"dataset_name":"acsdt5y2022","geoid":"0500000US06001","source_name":"acsdt5y2022-b01001.dat","values":{"b01001_001":1663823,"b01001_002":826561,"b01001_003":45925,"b01001_004":47576,"b01001_005":48382,"b01001_006":28544,"b01001_007":20330,"b01001_008":10059,"b01001_009":9868,"b01001_010":28353,"b01001_011":62380,"b01001_012":72338,"b01001_013":69473,"b01001_014":62112,"b01001_015":57592,"b01001_016":55360,"b01001_017":52792,"b01001_018":20592,"b01001_019":27469,"b01001_020":16861,"b01001_021":19794,"b01001_022":30165,"b01001_023":17775,"b01001_024":12014,"b01001_025":10807,"b01001_026":837262,"b01001_027":44062,"b01001_028":44832,"b01001_029":46225,"b01001_030":27281,"b01001_031":20424,"b01001_032":9802,"b01001_033":10241,"b01001_034":28031,"b01001_035":61402,"b01001_036":70642,"b01001_037":67277,"b01001_038":59739,"b01001_039":55980,"b01001_040":53250,"b01001_041":53257,"b01001_042":21514,"b01001_043":27480,"b01001_044":17165,"b01001_045":23991,"b01001_046":35310,"b01001_047":23707,"b01001_048":15582,"b01001_049":20068}}}],"pageInfo":{"hasNextPage":false}}}`,
		},
		{
			name:              "census_values combined filters - table and geoid_prefix",
			query:             `query { census_values(first:10, where:{table:"b01001", geoid_prefix:"1400000US0600140"}) { edges { node { geoid dataset_name values table { table_name } } } pageInfo { hasNextPage endCursor } } }`,
			vars:              vars,
			selector:          "census_values.edges.#.node.geoid",
			selectExpectCount: 10,
			f: func(t *testing.T, jj string) {
				tables := gjson.Get(jj, "census_values.edges.#.node.table.table_name").Array()
				geoids := gjson.Get(jj, "census_values.edges.#.node.geoid").Array()
				for _, tbl := range tables {
					assert.Equal(t, "b01001", tbl.String(), "all results should be from b01001 table")
				}
				for _, geoid := range geoids {
					assert.Contains(t, geoid.String(), "1400000US0600140", "all geoids should start with the prefix")
				}
			},
		},
		{
			name:   "census_values combined filters - all filters",
			query:  `query { census_values(where:{dataset:"acsdt5y2022", table:"b01001", geoid:"1400000US06001403000"}) { edges { node { geoid dataset_name source_name values table { table_name } } } pageInfo { hasNextPage } } }`,
			vars:   vars,
			expect: `{"census_values":{"edges":[{"node":{"dataset_name":"acsdt5y2022","geoid":"1400000US06001403000","source_name":"acsdt5y2022-b01001.dat","table":{"table_name":"b01001"},"values":{"b01001_001":4758,"b01001_002":1875,"b01001_003":75,"b01001_004":37,"b01001_005":100,"b01001_006":8,"b01001_007":40,"b01001_008":0,"b01001_009":0,"b01001_010":133,"b01001_011":275,"b01001_012":113,"b01001_013":325,"b01001_014":174,"b01001_015":162,"b01001_016":96,"b01001_017":94,"b01001_018":35,"b01001_019":27,"b01001_020":29,"b01001_021":31,"b01001_022":93,"b01001_023":21,"b01001_024":0,"b01001_025":7,"b01001_026":2883,"b01001_027":216,"b01001_028":123,"b01001_029":203,"b01001_030":74,"b01001_031":34,"b01001_032":0,"b01001_033":17,"b01001_034":6,"b01001_035":429,"b01001_036":258,"b01001_037":320,"b01001_038":137,"b01001_039":243,"b01001_040":167,"b01001_041":222,"b01001_042":32,"b01001_043":108,"b01001_044":10,"b01001_045":60,"b01001_046":45,"b01001_047":91,"b01001_048":20,"b01001_049":68}}}],"pageInfo":{"hasNextPage":false}}}`,
		},
		{
			name:   "census_values no results with non-existent geoid",
			query:  `query { census_values(where:{geoid:"9999999999999"}) { edges { node { geoid dataset_name values } } pageInfo { hasNextPage } } }`,
			vars:   vars,
			expect: `{"census_values":{"edges":[],"pageInfo":{"hasNextPage":false}}}`,
		},
		{
			name:   "census_values no results with non-existent table",
			query:  `query { census_values(where:{table:"nonexistent"}) { edges { node { geoid dataset_name values } } pageInfo { hasNextPage } } }`,
			vars:   vars,
			expect: `{"census_values":{"edges":[],"pageInfo":{"hasNextPage":false}}}`,
		},
		{
			name:  "census_values pagination test",
			query: `query { census_values(first:5, where:{geoid_prefix:"1400000US0600140"}) { edges { node { geoid } cursor } pageInfo { hasNextPage endCursor } } }`,
			vars:  vars,
			f: func(t *testing.T, jj string) {
				hasNextPage := gjson.Get(jj, "census_values.pageInfo.hasNextPage").Bool()
				endCursor := gjson.Get(jj, "census_values.pageInfo.endCursor").String()
				assert.True(t, hasNextPage, "should have next page")
				assert.NotEmpty(t, endCursor, "should have end cursor")
				edges := gjson.Get(jj, "census_values.edges").Array()
				assert.Equal(t, 5, len(edges), "should return 5 results")
			},
		},
	}
	queryTestcases(t, c, testcases)
}
func testIntersectionArea(t *testing.T, a []gjson.Result, expectCount int, expectGeometryArea float64, expectIntersectionArea float64) {
	// Only count each geometry once
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
}
