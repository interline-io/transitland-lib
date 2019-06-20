package enums

import "testing"

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
