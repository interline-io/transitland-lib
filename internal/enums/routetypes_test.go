package enums

import (
	"fmt"
	"sort"
	"testing"
)

func TestGetPrimitiveRouteType(t *testing.T) {
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
		{900, 2, true},
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
		rt, ok := GetPrimitiveRouteType((i.code))
		result := rt.Code
		if ok != i.ok {
			t.Errorf("got %t expect %t", ok, i.ok)
		}
		if result != i.primitive {
			t.Errorf("got %d expect %d", result, i.primitive)
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
			t.Errorf("got %t expect %t", ok, i.ok)
		}
		if ok && rt.Code != i.code {
			t.Errorf("got %d expect %d", rt.Code, i.code)
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
		fmt.Println(rets)
		sort.Ints(rets)
		sort.Ints(testcase.rets)
		if len(rets) != len(testcase.rets) {
			t.Errorf("got len %d expect len %d", len(rets), len(testcase.rets))
		} else {
			for i := range rets {
				if rets[i] != testcase.rets[i] {
					t.Errorf("got %d expect %d", rets[i], testcase.rets[i])
				}
			}
		}
	}
}
