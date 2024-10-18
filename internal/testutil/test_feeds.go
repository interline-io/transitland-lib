package testutil

import "github.com/interline-io/transitland-lib/internal/testpath"

// ExampleDir .
var ExampleDir = ReaderTester{
	URL: testpath.RelPath("testdata/example"),
	Counts: map[string]int{
		"agency.txt":          1,
		"routes.txt":          5,
		"trips.txt":           11,
		"stops.txt":           9,
		"stop_times.txt":      28,
		"shapes.txt":          9,
		"calendar.txt":        3, // this will be 3 because the simple DirectCopy copier does not filter out generated entities
		"calendar_dates.txt":  2,
		"feed_info.txt":       1,
		"fare_rules.txt":      4,
		"fare_attributes.txt": 2,
		"frequency.txt":       11,
		"transfers.txt":       0,
	},
	EntityIDs: map[string][]string{
		"agency.txt":          {"DTA"},
		"routes.txt":          {"AB", "BFC", "STBA", "CITY", "AAMV"},
		"trips.txt":           {"AB1", "AB2", "STBA", "CITY1", "CITY2", "BFC1", "BFC2", "AAMV1", "AAMV2", "AAMV3", "AAMV4"},
		"stops.txt":           {"FUR_CREEK_RES", "BULLFROG"}, // partial
		"shapes.txt":          {"ok", "a", "c"},
		"calendar.txt":        {"FULLW", "WE"},
		"fare_attributes.txt": {"p", "a"},
	},
	DirSHA1: "7a5c69b5466746213eb3cb6d907a7004073eca4d",
}

// ExampleZip .
var ExampleZip = ReaderTester{
	URL:     testpath.RelPath("testdata/example.zip"),
	SHA1:    "ce0a38dd6d4cfdac6aebe003181b6b915390a3b8",
	DirSHA1: "7a5c69b5466746213eb3cb6d907a7004073eca4d",
	Size:    4197,
	Counts: map[string]int{
		"agency.txt":          1,
		"routes.txt":          5,
		"trips.txt":           11,
		"stops.txt":           9,
		"stop_times.txt":      28,
		"shapes.txt":          9,
		"calendar.txt":        3, // 1 generated
		"calendar_dates.txt":  2,
		"feed_info.txt":       1,
		"fare_rules.txt":      4,
		"fare_attributes.txt": 2,
		"frequency.txt":       11,
		"transfers.txt":       0,
	},
	EntityIDs: map[string][]string{
		"agency.txt":          {"DTA"},
		"routes.txt":          {"AB", "BFC", "STBA", "CITY", "AAMV"},
		"trips.txt":           {"AB1", "AB2", "STBA", "CITY1", "CITY2", "BFC1", "BFC2", "AAMV1", "AAMV2", "AAMV3", "AAMV4"},
		"stops.txt":           {"FUR_CREEK_RES", "BULLFROG"}, // partial
		"shapes.txt":          {"ok", "a", "c"},
		"calendar.txt":        {"FULLW", "WE"},
		"fare_attributes.txt": {"p", "a"},
	},
}

// ExampleZipNestedDir .
var ExampleZipNestedDir = ReaderTester{
	URL: testpath.RelPath("testdata/example-nested-dir.zip#example-nested-dir/example"),
}

var ExampleZipNestedTwoFeeds1 = ReaderTester{
	URL: testpath.RelPath("testdata/example-nested-two-feeds.zip#example1"),
}

var ExampleZipNestedTwoFeeds2 = ReaderTester{
	URL: testpath.RelPath("testdata/example-nested-two-feeds.zip#example2"),
}

// ExampleZipNestedZip .
var ExampleZipNestedZip = ReaderTester{
	URL: testpath.RelPath("testdata/example-nested-zip.zip#example-nested-zip/example.zip"),
}

// ExampleFeedBART - BART test feed
var ExampleFeedBART = ReaderTester{
	URL: testpath.RelPath("testdata/external/bart.zip"),
	Counts: map[string]int{
		"agency.txt":          1,
		"calendar.txt":        3,
		"calendar_dates.txt":  12,
		"fare_attributes.txt": 170,
		"fare_rules.txt":      2304,
		"feed_info.txt":       1,
		"routes.txt":          6,
		"shapes.txt":          25074,
		"stop_times.txt":      33167,
		"stops.txt":           50,
		"transfers.txt":       8,
		"trips.txt":           2525},
	EntityIDs: map[string][]string{
		"agency.txt":          {"BART"},
		"calendar.txt":        {"WKDY", "SAT", "SUN"},
		"fare_attributes.txt": {"50", "51", "52", "53", "54", "56", "57", "58", "59", "60"},
		"routes.txt": {"01",
			"03",
			"05",
			"07",
			"11",
			"19"},
		"shapes.txt": {"01_shp",
			"02_shp",
			"03_shp",
			"04_shp",
			"05_shp",
			"06_shp",
			"07_shp",
			"08_shp",
			"11_shp",
			"12_shp"},
		"stops.txt": {"12TH",
			"16TH",
			"19TH",
			"19TH_N",
			"24TH",
			"ANTC",
			"ASHB",
			"BALB",
			"BAYF",
			"CAST"},
		"trips.txt": {"3610403WKDY",
			"3730559SAT",
			"3650800SUN",
			"3630418WKDY",
			"3750558SAT",
			"3670758SUN",
			"3650433WKDY",
			"3770618SAT",
			"3690818SUN",
			"3670448WKDY"}}}

// ExampleFeedCaltrain - Caltrain test feed
var ExampleFeedCaltrain = ReaderTester{
	URL: testpath.RelPath("testdata/external/caltrain.zip"),
	Counts: map[string]int{
		"agency.txt":          1,
		"calendar.txt":        27, // 3 - 24 generated
		"calendar_dates.txt":  36,
		"fare_attributes.txt": 6,
		"fare_rules.txt":      216,
		"feed_info.txt":       0,
		"routes.txt":          6,
		"shapes.txt":          3008,
		"stop_times.txt":      2853,
		"stops.txt":           64,
		"transfers.txt":       0,
		"trips.txt":           185},
}

// MBTA
var ExampleFeedMBTA = ReaderTester{
	URL: testpath.RelPath("testdata/external/mbta.zip"),
	Counts: map[string]int{
		"agency.txt":         2,
		"calendar.txt":       122,
		"calendar_dates.txt": 31,
		"feed_info.txt":      1,
		"routes.txt":         216,
		"shapes.txt":         897,
		"stop_times.txt":     1291946,
		"stops.txt":          9838,
		"transfers.txt":      1834,
		"trips.txt":          55166},
	EntityIDs: map[string][]string{
		"agency.txt": {
			"3",
			"1"},
		"calendar.txt": {
			"BUS319-1-Wdy-02",
			"BUS319-2-Wdy-02",
			"BUS319-3-Sa-02",
			"BUS319-4-Su-02",
			"BUS319-5-Wdy-02",
			"BUS319-6-Sa-02",
			"BUS319-7-Su-02",
			"BUS319-8-Wdy-02",
			"BUS319-9-Sa-02",
			"BUS319-A-Su-02"},
		"routes.txt": {
			"Red",
			"Mattapan",
			"Orange",
			"Green-B",
			"Green-C",
			"Green-D",
			"Green-E",
			"Blue",
			"741",
			"742"},
		"shapes.txt": {
			"010058",
			"010070",
			"040033",
			"040034",
			"040037",
			"040038",
			"050036",
			"050037",
			"070070",
			"070071"},
		"stops.txt": {
			"1",
			"10",
			"10000",
			"10003",
			"10005",
			"10006",
			"10007",
			"10008",
			"10009",
			"10010"},
		"trips.txt": {
			"40667713",
			"40667714",
			"40667720",
			"40667721",
			"40667723",
			"40667724",
			"40667728",
			"40667730",
			"40667731",
			"40667734"},
	},
}

var ExampleFeedCDMX = ReaderTester{
	URL: testpath.RelPath("testdata/external/cdmx.zip"),
	Counts: map[string]int{
		"agency.txt":      8,
		"calendar.txt":    99,
		"frequencies.txt": 1741,
		"routes.txt":      145,
		"shapes.txt":      345,
		"stop_times.txt":  70714,
		"stops.txt":       6021,
		"trips.txt":       1741},
	EntityIDs: map[string][]string{
		"agency.txt": {
			"CC",
			"MB",
			"METRO",
			"NCC",
			"RTP",
			"RTP_ESP",
			"STE",
			"SUB"},
		"calendar.txt": {
			"14741",
			"16092",
			"16203",
			"16233",
			"26656",
			"26949",
			"28960",
			"36238",
			"36384",
			"36479"},
		"routes.txt": {
			"ROUTE_132162",
			"ROUTE_132207",
			"ROUTE_134667",
			"ROUTE_18226",
			"ROUTE_36490",
			"ROUTE_36644",
			"ROUTE_36712",
			"ROUTE_36713",
			"ROUTE_136284",
			"ROUTE_136285"},
		"shapes.txt": {
			"14816",
			"14817",
			"14818",
			"14819",
			"14820",
			"14821",
			"14822",
			"14823",
			"14824",
			"14825"},
		"stops.txt": {
			"136300",
			"136299",
			"28503",
			"28800",
			"28805",
			"28812",
			"28830",
			"28832",
			"28833",
			"28963"},
		"trips.txt": {
			"14743",
			"14840",
			"14841",
			"14842",
			"14843",
			"14844",
			"14845",
			"14846",
			"14848",
			"15171"},
	},
}

// ExternalTestFeed returns an external test feed by filename.
func ExternalTestFeed(key string) (ReaderTester, bool) {
	a, ok := ExternalTestFeeds[key]
	return a, ok
}

// ExternalTestFeeds is a collection of known data about the external test feeds.
var ExternalTestFeeds = map[string]ReaderTester{
	"example.zip":  ExampleZip,
	"bart.zip":     ExampleFeedBART,
	"caltrain.zip": ExampleFeedCaltrain,
	"cdmx.zip":     ExampleFeedCDMX,
	"mbta.zip":     ExampleFeedMBTA,
}
