package builders

import "testing"

func Test_pointsGeohash(t *testing.T) {
	p1 := point{Lon: -122.407264, Lat: 37.788081}                // sf
	p2 := point{Lon: -122.268905, Lat: 37.806528}                // oakland
	p3 := point{Lon: -121.911163, Lat: 37.341775}                // sj
	p4 := point{Lon: -121.618652, Lat: 39.147102}                // sac
	p5 := point{Lon: -122.384948, Lat: 37.322120}                // salinas
	p6 := point{Lon: -120.671081, Lat: 35.281500}                // slo
	p7 := point{Lon: -122.3272705078125, Lat: 47.59505101193038} // seattle
	testcases := []struct {
		points []point
		expect string
		minc   uint
	}{
		{[]point{p1}, "9q8yyx1g", 3},
		{[]point{p1, p2}, "9q9p", 3},
		{[]point{p1, p2, p3}, "9q9", 3},
		{[]point{p1, p2, p3, p4, p5, p6}, "9q9", 3},
		{[]point{p4, p6}, "9q9", 3},
		{[]point{p1, p7}, "9r", 2},
		{[]point{p1, p7}, "", 3}, // can't generate with at least 3 characters
	}
	for _, tc := range testcases {
		t.Run("", func(t *testing.T) {
			if gh := pointsGeohash(tc.points, tc.minc, 8); gh != tc.expect {
				t.Errorf("got '%s' expect '%s'", gh, tc.expect)
			}
		})
	}
}

func Test_filterName(t *testing.T) {
	testcases := []struct {
		value  string
		expect string
	}{
		{"hello", "hello"},
		{"he ll o", "hello"},
		{"he.l.l.o", "hello"},
		{"hello 1 2 3", "hello123"},
		{"San José", "sanjosé"},
		{"你好", "你好"},
	}
	for _, tc := range testcases {
		t.Run(tc.value, func(t *testing.T) {
			if g := filterName(tc.value); g != tc.expect {
				t.Errorf("got '%s' expect '%s'", g, tc.expect)
			}
		})
	}
}
