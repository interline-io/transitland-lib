package testutil

// TestFeeds .
var TestFeeds = map[string]ExpectEntities{
	"example": ExpectEntities{
		URL:                "../testdata/example",
		AgencyCount:        1,
		RouteCount:         5,
		TripCount:          11,
		StopCount:          9,
		StopTimeCount:      28,
		ShapeCount:         3,
		CalendarCount:      2,
		CalendarDateCount:  2,
		FeedInfoCount:      1,
		FareRuleCount:      4,
		FareAttributeCount: 2,
		FrequencyCount:     11,
		TransferCount:      0,
		ExpectAgencyIDs:    []string{"DTA"},
		ExpectRouteIDs:     []string{"AB", "BFC", "STBA", "CITY", "AAMV"},
		ExpectTripIDs:      []string{"AB1", "AB2", "STBA", "CITY1", "CITY2", "BFC1", "BFC2", "AAMV1", "AAMV2", "AAMV3", "AAMV4"},
		ExpectStopIDs:      []string{"FUR_CREEK_RES", "BULLFROG"}, // partial
		ExpectShapeIDs:     []string{"ok", "a", "c"},
		ExpectCalendarIDs:  []string{"FULLW", "WE"},
		ExpectFareIDs:      []string{"p", "a"},
	},
	"bart": ExpectEntities{
		URL:                "../testdata/external/bart.zip",
		AgencyCount:        1,
		RouteCount:         6,
		TripCount:          2525,
		StopCount:          50,
		StopTimeCount:      33167,
		ShapeCount:         12,
		CalendarCount:      3,
		CalendarDateCount:  12,
		FeedInfoCount:      1,
		FareRuleCount:      2304,
		FareAttributeCount: 170,
		FrequencyCount:     0,
		TransferCount:      8,
		ExpectAgencyIDs:    []string{"BART"},
		ExpectRouteIDs:     []string{"05", "07", "11", "19", "01", "03"},
		ExpectTripIDs:      []string{"3610403WKDY", "5152308WKDY", "8051436SAT", "3211529WKDY", "2232328SAT"}, // partial
		ExpectStopIDs: []string{
			"19TH", "ANTC", "MCAR", "POWL", "RICH", "SSAN", "WDUB", "16TH", "24TH", "CIVC", "SBRN", "UCTY", "WCRK", "12TH", "DELN", "GLEN", "HAYW", "PHIL", "ROCK", "SANL", "DUBL", "DALY", "DBRK", "MLBR", "ORIN", "SHAY", "BALB", "LAFY", "LAKE", "MCAR_S", "NBRK", "NCON", "ASHB", "COLM", "EMBR", "PITT", "WARM", "BAYF", "CONC", "FRMT", "MONT", "WOAK", "COLS", "CAST", "PLZA", "FTVL", "OAKL", "PCTR", "SFIA", "19TH_N"}, // partial
		ExpectShapeIDs:    []string{"06_shp", "07_shp", "08_shp", "11_shp", "20_shp", "02_shp", "04_shp", "05_shp", "12_shp", "19_shp", "01_shp", "03_shp"},
		ExpectCalendarIDs: []string{"WKDY", "SAT", "SUN"},
		ExpectFareIDs:     []string{"73", "77", "93", "202", "207", "57", "99", "52", "117", "120", "179", "79", "89", "133", "141", "204", "53", "59", "80", "87", "92", "131", "132", "173", "178", "183", "190", "260", "71", "82", "107", "124", "222", "56", "106", "140", "198", "214", "60", "231", "86", "112", "143", "148", "242", "50", "65", "102", "257", "100", "111", "163", "225", "76", "95", "145", "166", "157", "209", "74", "122", "150", "182", "199", "63", "137", "139", "153", "167", "170", "180", "70", "113", "146", "154", "156", "200", "206", "215", "232", "51", "118", "130", "193", "197", "104", "195", "227", "58", "72", "185", "223", "246", "88", "98", "108", "109", "127", "149", "172", "192", "61", "81", "91", "103", "123", "186", "208", "66", "142", "161", "203", "212", "241", "254", "114", "229", "54", "116", "121", "125", "138", "219", "333", "64", "84", "119", "220", "221", "97", "101", "158", "162", "238", "249", "78", "105", "211", "226", "258", "83", "90", "96", "115", "144", "152", "169", "205", "75", "134", "135", "151", "160", "216", "233", "245", "62", "69", "85", "94", "126", "128", "147", "165", "234", "273", "110", "129", "136", "189"},
	},
}

// TestFeed .
func TestFeed(key string) (ExpectEntities, bool) {
	a, ok := TestFeeds[key]
	return a, ok
}
