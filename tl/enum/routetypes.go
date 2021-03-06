package enum

// RouteType contains details on each possible route_type
type RouteType struct {
	Code     int
	Name     string
	Category string
	Parent   int
}

// routeTypes is the list of all known extended route_types.
// See https://developers.google.com/transit/gtfs/reference/extended-route-types
// All types are mapped back to a basic GTFS primitive.
// Those that don't fit well are mapped to bus for compatibility.
var routeTypes = []RouteType{
	{Code: 0, Name: "Tram"},
	{Code: 1, Name: "Metro"},
	{Code: 2, Name: "Rail"},
	{Code: 3, Name: "Bus"},
	{Code: 4, Name: "Ferry"},
	{Code: 5, Name: "Cablecar"},
	{Code: 6, Name: "Gondola"},
	{Code: 7, Name: "Funicular"},
	// {Code: 11, Name: "Trolleybus"},
	// {Code: 12, Name: "Monorail"},

	{Code: 100, Name: "Railway Service", Parent: 2},
	{Code: 101, Name: "High Speed Rail Service", Parent: 2},
	{Code: 102, Name: "Long Distance Trains", Parent: 2},
	{Code: 103, Name: "Inter Regional Rail Service", Parent: 2},
	{Code: 104, Name: "Car Transport Rail Service", Parent: 2},
	{Code: 105, Name: "Sleeper Rail Service", Parent: 2},
	{Code: 106, Name: "Regional Rail Service", Parent: 2},
	{Code: 107, Name: "Tourist Railway Service", Parent: 2},
	{Code: 108, Name: "Rail Shuttle (Within Complex)", Parent: 2},
	{Code: 109, Name: "Suburban Railway", Parent: 2},
	{Code: 110, Name: "Replacement Rail Service", Parent: 2},
	{Code: 111, Name: "Special Rail Service", Parent: 2},
	{Code: 112, Name: "Lorry Transport Rail Service", Parent: 2},
	{Code: 113, Name: "All Rail Services", Parent: 2},
	{Code: 114, Name: "Cross-Country Rail Service", Parent: 2},
	{Code: 115, Name: "Vehicle Transport Rail Service", Parent: 2},
	{Code: 116, Name: "Rack and Pinion Railway", Parent: 2},
	{Code: 117, Name: "Additional Rail Service", Parent: 2},

	{Code: 200, Name: "Coach Service", Parent: 3},
	{Code: 201, Name: "International Coach Service", Parent: 3},
	{Code: 202, Name: "National Coach Service", Parent: 3},
	{Code: 203, Name: "Shuttle Coach Service", Parent: 3},
	{Code: 204, Name: "Regional Coach Service", Parent: 3},
	{Code: 205, Name: "Special Coach Service", Parent: 3},
	{Code: 206, Name: "Sightseeing Coach Service", Parent: 3},
	{Code: 207, Name: "Tourist Coach Service", Parent: 3},
	{Code: 208, Name: "Commuter Coach Service", Parent: 3},
	{Code: 209, Name: "All Coach Services", Parent: 3},

	{Code: 300, Name: "Suburban Railway Service", Parent: 2},

	{Code: 400, Name: "Urban Railway Service", Parent: 1},
	{Code: 401, Name: "Metro Service", Parent: 1},
	{Code: 402, Name: "Underground Service", Parent: 1},
	{Code: 403, Name: "Urban Railway Service", Parent: 1},
	{Code: 404, Name: "All Urban Railway Services", Parent: 1},
	{Code: 405, Name: "Monorail", Parent: 1},

	{Code: 700, Name: "Bus Service", Parent: 3},
	{Code: 701, Name: "Regional Bus Service", Parent: 3},
	{Code: 702, Name: "Express Bus Service", Parent: 3},
	{Code: 703, Name: "Stopping Bus Service", Parent: 3},
	{Code: 704, Name: "Local Bus Service", Parent: 3},
	{Code: 705, Name: "Night Bus Service", Parent: 3},
	{Code: 706, Name: "Post Bus Service", Parent: 3},
	{Code: 707, Name: "Special Needs Bus", Parent: 3},
	{Code: 708, Name: "Mobility Bus Service", Parent: 3},
	{Code: 709, Name: "Mobility Bus for Registered Disabled", Parent: 3},
	{Code: 710, Name: "Sightseeing Bus", Parent: 3},
	{Code: 711, Name: "Shuttle Bus", Parent: 3},
	{Code: 712, Name: "School Bus", Parent: 3},
	{Code: 713, Name: "School and Public Service Bus", Parent: 3},
	{Code: 714, Name: "Rail Replacement Bus Service", Parent: 3},
	{Code: 715, Name: "Demand and Response Bus Service", Parent: 3},
	{Code: 716, Name: "All Bus Services", Parent: 3},
	{Code: 717, Name: "Share Taxi Service", Parent: 3},

	{Code: 800, Name: "Trolleybus Service", Parent: 3},

	{Code: 900, Name: "Tram Service", Parent: 0},
	{Code: 901, Name: "City Tram Service", Parent: 0},
	{Code: 902, Name: "Local Tram Service", Parent: 0},
	{Code: 903, Name: "Regional Tram Service", Parent: 0},
	{Code: 904, Name: "Sightseeing Tram Service", Parent: 0},
	{Code: 905, Name: "Shuttle Tram Service", Parent: 0},
	{Code: 906, Name: "All Tram Services", Parent: 0},
	{Code: 907, Name: "Cable Tram", Parent: 0},

	{Code: 1000, Name: "Water Transport Service", Parent: 4},
	{Code: 1001, Name: "International Car Ferry Service", Parent: 4},
	{Code: 1002, Name: "National Car Ferry Service", Parent: 4},
	{Code: 1003, Name: "Regional Car Ferry Service", Parent: 4},
	{Code: 1004, Name: "Local Car Ferry Service", Parent: 4},
	{Code: 1005, Name: "International Passenger Ferry Service", Parent: 4},
	{Code: 1006, Name: "National Passenger Ferry Service", Parent: 4},
	{Code: 1007, Name: "Regional Passenger Ferry Service", Parent: 4},
	{Code: 1008, Name: "Local Passenger Ferry Service", Parent: 4},
	{Code: 1009, Name: "Post Boat Service", Parent: 4},
	{Code: 1010, Name: "Train Ferry Service", Parent: 4},
	{Code: 1011, Name: "Road-Link Ferry Service", Parent: 4},
	{Code: 1012, Name: "Airport-Link Ferry Service", Parent: 4},
	{Code: 1013, Name: "Car High-Speed Ferry Service", Parent: 4},
	{Code: 1014, Name: "Passenger High-Speed Ferry Service", Parent: 4},
	{Code: 1015, Name: "Sightseeing Boat Service", Parent: 4},
	{Code: 1016, Name: "School Boat", Parent: 4},
	{Code: 1017, Name: "Cable-Drawn Boat Service", Parent: 4},
	{Code: 1018, Name: "River Bus Service", Parent: 4},
	{Code: 1019, Name: "Scheduled Ferry Service", Parent: 4},
	{Code: 1020, Name: "Shuttle Ferry Service", Parent: 4},
	{Code: 1021, Name: "All Water Transport Services", Parent: 4},

	{Code: 1100, Name: "Air Service", Parent: 1700},
	{Code: 1101, Name: "International Air Service", Parent: 1100},
	{Code: 1102, Name: "Domestic Air Service", Parent: 1100},
	{Code: 1103, Name: "Intercontinental Air Service", Parent: 1100},
	{Code: 1104, Name: "Domestic Scheduled Air Service", Parent: 1100},
	{Code: 1105, Name: "Shuttle Air Service", Parent: 1100},
	{Code: 1106, Name: "Intercontinental Charter Air Service", Parent: 1100},
	{Code: 1107, Name: "International Charter Air Service", Parent: 1100},
	{Code: 1108, Name: "Round-Trip Charter Air Service", Parent: 1100},
	{Code: 1109, Name: "Sightseeing Air Service", Parent: 1100},
	{Code: 1110, Name: "Helicopter Air Service", Parent: 1100},
	{Code: 1111, Name: "Domestic Charter Air Service", Parent: 1100},
	{Code: 1112, Name: "Schengen-Area Air Service", Parent: 1100},
	{Code: 1113, Name: "Airship Service", Parent: 1100},
	{Code: 1114, Name: "All Air Services", Parent: 1100},

	{Code: 1200, Name: "Ferry Service", Parent: 4},

	{Code: 1300, Name: "Aerial Lift Service", Parent: 6},
	{Code: 1301, Name: "Telecabin Service", Parent: 6},
	{Code: 1302, Name: "Cable Car Service", Parent: 6},
	{Code: 1303, Name: "Elevator Service", Parent: 6},
	{Code: 1304, Name: "Chair Lift Service", Parent: 6},
	{Code: 1305, Name: "Drag Lift Service", Parent: 6},
	{Code: 1306, Name: "Small Telecabin Service", Parent: 6},
	{Code: 1307, Name: "All Telecabin Services", Parent: 6},

	{Code: 1400, Name: "Funicular Service", Parent: 7},
	{Code: 1401, Name: "Funicular Service", Parent: 7},
	{Code: 1402, Name: "All Funicular Service", Parent: 7},

	{Code: 1500, Name: "Taxi Service", Parent: 1700},
	{Code: 1501, Name: "Communal Taxi Service", Parent: 1500},
	{Code: 1502, Name: "Water Taxi Service", Parent: 1500},
	{Code: 1503, Name: "Rail Taxi Service", Parent: 1500},
	{Code: 1504, Name: "Bike Taxi Service", Parent: 1500},
	{Code: 1505, Name: "Licensed Taxi Service", Parent: 1500},
	{Code: 1506, Name: "Private Hire Service Vehicle", Parent: 1500},
	{Code: 1507, Name: "All Taxi Services", Parent: 1500},

	{Code: 1600, Name: "Self Drive", Parent: 1700},
	{Code: 1601, Name: "Hire Car", Parent: 1600},
	{Code: 1602, Name: "Hire Van", Parent: 1600},
	{Code: 1603, Name: "Hire Motorbike", Parent: 1600},
	{Code: 1604, Name: "Hire Cycle", Parent: 1600},

	{Code: 1700, Name: "Miscellaneous Service", Parent: 3}, // Convert all unknown to Bus
	{Code: 1701, Name: "Cable Car", Parent: 5},
	{Code: 1702, Name: "Horse-drawn Carriage", Parent: 1700},
}

var routeTypesMap map[int]RouteType

func init() {
	routeTypesMap = map[int]RouteType{}
	for _, rt := range routeTypes {
		routeTypesMap[rt.Code] = rt
	}
}

// GetRouteType returns the details for a given route_type value.
func GetRouteType(code int) (RouteType, bool) {
	rt, ok := routeTypesMap[code]
	return rt, ok
}

// GetBasicRouteType returns the closest approximate basic route_type for an extended route_type.
func GetBasicRouteType(code int) (RouteType, bool) {
	parents := map[int]int{}
	for _, i := range routeTypes {
		if i.Code > 7 {
			parents[i.Code] = i.Parent
		}
	}
	for {
		code2, ok := parents[code]
		if !ok || code == code2 {
			break
		}
		code = code2
	}
	return GetRouteType(code)
}

func getRouteChildren(code int) []RouteType {
	children := map[int][]int{}
	for _, i := range routeTypes {
		if i.Code > 7 {
			children[i.Parent] = append(children[i.Parent], i.Code)
		}
	}
	queue := []int{}
	visited := map[int]bool{}
	if rt, ok := GetRouteType(code); ok {
		queue = append(queue, rt.Code)
	}
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		visited[item] = true
		for _, child := range children[item] {
			if !visited[child] {
				queue = append(queue, child)
			}
		}
	}
	ret := []RouteType{}
	for _, rt := range routeTypes {
		if _, ok := visited[rt.Code]; ok {
			ret = append(ret, rt)
		}
	}
	return ret
}
