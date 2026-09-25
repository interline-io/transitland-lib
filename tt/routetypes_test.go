package tt

import (
	"encoding/json"
	"sort"
	"testing"
)

func TestGetBasicRouteType(t *testing.T) {
	tests := []struct {
		code      int
		primitive int
		ok        bool
	}{
		{0, 0, true},
		{1, 1, true},
		{2, 2, true},
		{100, 2, true},
		{200, 3, true},
		{300, 2, true},
		{400, 1, true},
		{405, 12, true},
		{700, 3, true},
		{800, 11, true},
		{900, 0, true},
		{901, 0, true},
		{1000, 4, true},
		{1200, 4, true},
		{1300, 6, true},
		// map back to bus
		{1100, 3, true},
		{1101, 3, true},
		{1700, 3, true},
		{1600, 3, true},
		// missing
		{100000, 0, false},
	}
	for _, i := range tests {
		rt, ok := GetBasicRouteType((i.code))
		result := rt.Code
		if ok != i.ok {
			t.Errorf("code %d: got %t expect %t", i.code, ok, i.ok)
		}
		if result != i.primitive {
			t.Errorf("code %d: got %d expect %d", i.code, result, i.primitive)
		}
	}
}

func Test_RouteTypeJson(t *testing.T) {
	// This test exists only to dump the route type list to json
	// for use in other applications.
	jj, err := json.Marshal(routeTypes)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(jj))
}

func TestGetRouteType(t *testing.T) {
	tests := []struct {
		code   int
		parent int
		ok     bool
	}{
		{1, 0, true},
		{4, 0, true},
		{100, 2, true},
		{100000, 0, false},
	}
	for _, i := range tests {
		rt, ok := GetRouteType(i.code)
		if ok != i.ok {
			t.Errorf("code %d: got %t expect %t", i.code, ok, i.ok)
		}
		if ok && rt.Code != i.code {
			t.Errorf("code %d: got %d expect %d", i.code, rt.Code, i.code)
		}
	}
}

func Test_getRouteChildren(t *testing.T) {
	tests := []struct {
		code int
		rets []int
	}{
		{0, []int{0, 900, 901, 902, 903, 904, 905, 906, 907}},
		{7, []int{7, 1400, 1401, 1402}},
		{12, []int{12, 405}},
		{1100, []int{1100, 1101, 1102, 1103, 1104, 1105, 1106, 1107, 1108, 1109, 1110, 1111, 1112, 1113, 1114}},
		{1700, []int{1100, 1101, 1102, 1103, 1104, 1105, 1106, 1107, 1108, 1109, 1110, 1111, 1112, 1113, 1114, 1500, 1501, 1502, 1503, 1504, 1505, 1506, 1507, 1600, 1601, 1602, 1603, 1604, 1700, 1702}}, // recursive
	}
	for _, testcase := range tests {
		rets := []int{}
		for _, i := range getRouteChildren(testcase.code) {
			rets = append(rets, i.Code)
		}
		sort.Ints(rets)
		sort.Ints(testcase.rets)
		if len(rets) != len(testcase.rets) {
			t.Errorf("code %d: got len %d expect len %d", testcase.code, len(rets), len(testcase.rets))
		} else {
			for i := range rets {
				if rets[i] != testcase.rets[i] {
					t.Errorf("code %d: got %d expect %d", testcase.code, rets[i], testcase.rets[i])
				}
			}
		}
	}
}

func TestBasicRouteType(t *testing.T) {
	tests := []struct {
		name   string
		code   int
		expect int
	}{
		{"tram stands for itself", 0, 0},
		{"metro stands for itself", 1, 1},
		{"rail stands for itself", 2, 2},
		{"bus stands for itself", 3, 3},
		{"ferry stands for itself", 4, 4},
		{"cable tram stands for itself", 5, 5},
		{"aerial lift stands for itself", 6, 6},
		{"funicular stands for itself", 7, 7},
		{"trolleybus stands for itself", 11, 11},
		{"monorail stands for itself", 12, 12},
		// The codes a real network actually publishes: Norway's feeds are almost
		// entirely extended types.
		{"railway service is rail", 100, 2},
		{"long distance trains are rail", 101, 2},
		{"metro service is metro", 401, 1},
		{"monorail is monorail, not metro", 405, 12},
		{"local bus is bus", 702, 3},
		{"night bus is bus", 705, 3},
		{"school bus is bus", 712, 3},
		{"trolleybus is trolleybus, not bus", 800, 11},
		{"tram service is tram", 902, 0},
		{"water transport is ferry", 1000, 4},
		{"national car ferry is ferry", 1008, 4},
		{"telecabin is aerial lift", 1300, 6},
		{"funicular service is funicular", 1400, 7},
		{"taxi resolves through miscellaneous to bus", 1500, 3},
		{"cable car is cable tram, not bus", 1701, 5},
		// A code outside the hierarchy stands for itself rather than reading as
		// tram, which is what a zero would mean.
		{"an unknown code is left alone", 9999, 9999},
		{"an unassigned basic code is left alone", 8, 8},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := BasicRouteType(tc.code); got != tc.expect {
				t.Errorf("BasicRouteType(%d) = %d, expected %d", tc.code, got, tc.expect)
			}
		})
	}
}
