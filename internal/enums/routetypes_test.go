package enums

import (
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
		{700, 3, true},
		{800, 3, true},
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

func TestGetRouteChildren(t *testing.T) {
	tests := []struct {
		code int
		rets []int
	}{
		{0, []int{0, 900, 901, 902, 903, 904, 905, 906, 907}},
		{7, []int{7, 1400, 1401, 1402}},
		{1100, []int{1100, 1101, 1102, 1103, 1104, 1105, 1106, 1107, 1108, 1109, 1110, 1111, 1112, 1113, 1114}},
		{1700, []int{1100, 1101, 1102, 1103, 1104, 1105, 1106, 1107, 1108, 1109, 1110, 1111, 1112, 1113, 1114, 1500, 1501, 1502, 1503, 1504, 1505, 1506, 1507, 1600, 1601, 1602, 1603, 1604, 1700, 1702}}, // recursive
	}
	for _, testcase := range tests {
		rets := []int{}
		for _, i := range GetRouteChildren(testcase.code) {
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
