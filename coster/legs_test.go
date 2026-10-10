package coster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchLegRules_EmptyFields(t *testing.T) {
	// Without rule_priority, an empty field matches only values that no rule lists.
	c := newTestCoster(t, map[string]string{
		"areas.txt": `
area_id
zone1
zone2`,
		"stop_areas.txt": `
area_id,stop_id
zone1,a
zone2,b`,
		"fare_products.txt": `
fare_product_id,amount,currency
bus_fare,2.00,USD
other_fare,1.00,USD
rail_fare,4.00,USD
zone_fare,5.00,USD`,
		"fare_leg_rules.txt": `
network_id,from_area_id,to_area_id,fare_product_id
bus,,,bus_fare
,,,other_fare
rail,zone1,zone2,zone_fare
rail,,,rail_fare`,
	})
	tcs := []struct {
		name string
		leg  Leg
		want map[string]float64
	}{
		{"listed network", leg("bus1", "c", "d", at(8, 0), at(8, 20)), map[string]float64{"": 2.00}},
		{"unlisted network", leg("ferry", "c", "d", at(8, 0), at(8, 20)), map[string]float64{"": 1.00}},
		{"listed areas", leg("rail1", "a", "b", at(8, 0), at(8, 20)), map[string]float64{"": 5.00}},
		{"stops without areas", leg("rail1", "c", "d", at(8, 0), at(8, 20)), map[string]float64{"": 4.00}},
		// The zone rule needs zone2, and zone1 is listed, so the empty rule can't match this leg.
		{"listed departure area without a matching rule", leg("rail1", "a", "c", at(8, 0), at(8, 20)), map[string]float64{}},
		// An area that any rule lists can't match an empty field in any rule, even on another network.
		{"area listed on another network", leg("bus1", "a", "b", at(8, 0), at(8, 20)), map[string]float64{}},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, fareAmounts(t, c, journey(tc.leg)))
		})
	}
}

func TestMatchLegRules_StopInSeveralAreas(t *testing.T) {
	// An empty area field matches through any of a stop's areas that no rule lists.
	c := newTestCoster(t, map[string]string{
		"areas.txt": `
area_id
zone_x
zone_y
zone_z`,
		"stop_areas.txt": `
area_id,stop_id
zone_x,a
zone_z,a
zone_y,b`,
		"fare_products.txt": `
fare_product_id,amount,currency
zone_fare,5.00,USD
rail_fare,4.00,USD`,
		"fare_leg_rules.txt": `
network_id,from_area_id,to_area_id,fare_product_id
rail,zone_x,zone_y,zone_fare
rail,,,rail_fare`,
	})
	// No rule lists zone_z, so the empty rule matches the departure from a.
	assert.Equal(t, map[string]float64{"": 4.00}, fareAmounts(t, c, journey(leg("rail1", "a", "c", at(8, 0), at(8, 20)))))
	// A rule lists zone_y, so only the zone rule matches the arrival at b.
	assert.Equal(t, map[string]float64{"": 5.00}, fareAmounts(t, c, journey(leg("rail1", "a", "b", at(8, 0), at(8, 20)))))
}

func TestMatchLegRules_RulePriority(t *testing.T) {
	// With rule_priority, an empty field matches any value, and the highest priority wins.
	c := newTestCoster(t, map[string]string{
		"areas.txt": `
area_id
zone1
zone2`,
		"stop_areas.txt": `
area_id,stop_id
zone1,a
zone2,b`,
		"fare_products.txt": `
fare_product_id,amount,currency
flat_fare,1.00,USD
rail_fare,4.00,USD
zone_fare,5.00,USD`,
		"fare_leg_rules.txt": `
network_id,from_area_id,to_area_id,fare_product_id,rule_priority
,,,flat_fare,
rail,,,rail_fare,1
rail,zone1,zone2,zone_fare,2`,
	})
	tcs := []struct {
		name string
		leg  Leg
		want map[string]float64
	}{
		{"empty fields match any leg", leg("bus1", "a", "b", at(8, 0), at(8, 20)), map[string]float64{"": 1.00}},
		{"network rule outranks the empty rule", leg("rail1", "c", "d", at(8, 0), at(8, 20)), map[string]float64{"": 4.00}},
		{"area rule outranks the network rule", leg("rail1", "a", "b", at(8, 0), at(8, 20)), map[string]float64{"": 5.00}},
		{"listed area falls back to the network rule", leg("rail1", "a", "c", at(8, 0), at(8, 20)), map[string]float64{"": 4.00}},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, fareAmounts(t, c, journey(tc.leg)))
		})
	}
}

func TestMatchLegRules_StationAreas(t *testing.T) {
	// A platform is in its station's areas unless it has areas of its own.
	c := newTestCoster(t, map[string]string{
		"areas.txt": `
area_id
zone1
zone2
zone3`,
		"stop_areas.txt": `
area_id,stop_id
zone1,station
zone2,platform2
zone3,b`,
		"fare_products.txt": `
fare_product_id,amount,currency
zone1_zone3,3.00,USD
zone2_zone3,3.50,USD`,
		"fare_leg_rules.txt": `
network_id,from_area_id,to_area_id,fare_product_id
rail,zone1,zone3,zone1_zone3
rail,zone2,zone3,zone2_zone3`,
	})
	assert.Equal(t, map[string]float64{"": 3.00}, fareAmounts(t, c, journey(leg("rail1", "platform1", "b", at(8, 0), at(8, 20)))))
	assert.Equal(t, map[string]float64{"": 3.50}, fareAmounts(t, c, journey(leg("rail1", "platform2", "b", at(8, 0), at(8, 20)))))
}

// timeframeFeed has peak and off-peak timeframes on weekdays and one all-day timeframe on weekends.
var timeframeFeed = map[string]string{
	"calendar.txt": `
service_id,monday,tuesday,wednesday,thursday,friday,saturday,sunday,start_date,end_date
weekdays,1,1,1,1,1,0,0,20260101,20261231
weekends,0,0,0,0,0,1,1,20260101,20261231`,
	"timeframes.txt": `
timeframe_group_id,start_time,end_time,service_id
peak,07:00:00,09:00:00,weekdays
peak,16:00:00,18:00:00,weekdays
offpeak,00:00:00,07:00:00,weekdays
offpeak,09:00:00,16:00:00,weekdays
offpeak,18:00:00,24:00:00,weekdays
weekend,,,weekends`,
	"fare_products.txt": `
fare_product_id,amount,currency
peak_fare,5.00,USD
offpeak_fare,3.00,USD
weekend_fare,2.00,USD`,
}

func TestMatchLegRules_Timeframes(t *testing.T) {
	c := newTestCoster(t, timeframeFeed, map[string]string{
		"fare_leg_rules.txt": `
network_id,from_timeframe_group_id,fare_product_id
rail,peak,peak_fare
rail,offpeak,offpeak_fare
rail,weekend,weekend_fare`,
	})
	tcs := []struct {
		name string
		leg  Leg
		want float64
	}{
		{"weekday peak", leg("rail1", "c", "d", at(8, 0), at(8, 30)), 5.00},
		{"end_time is exclusive", leg("rail1", "c", "d", at(9, 0), at(9, 30)), 3.00},
		{"second peak range", leg("rail1", "c", "d", at(17, 59), at(18, 30)), 5.00},
		{"weekend", leg("rail1", "c", "d", oct(10, 8, 0), oct(10, 8, 30)), 2.00},
		// A Friday trip that departs after midnight departs on Saturday, the event's local date.
		{"after midnight", leg("rail1", "c", "d", oct(10, 0, 30), oct(10, 1, 0)), 2.00},
		// 08:00 in California is 11:00 at a stop in New York.
		{"stop timezone", leg("rail1", "east", "d", at(8, 0), at(8, 30)), 3.00},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, map[string]float64{"": tc.want}, fareAmounts(t, c, journey(tc.leg)))
		})
	}
}

func TestMatchLegRules_ToTimeframes(t *testing.T) {
	// to_timeframe_group_id matches the leg's arrival.
	c := newTestCoster(t, timeframeFeed, map[string]string{
		"fare_leg_rules.txt": `
network_id,to_timeframe_group_id,fare_product_id
rail,peak,peak_fare
rail,offpeak,offpeak_fare`,
	})
	assert.Equal(t, map[string]float64{"": 5.00}, fareAmounts(t, c, journey(leg("rail1", "c", "d", at(6, 50), at(7, 10)))))
	assert.Equal(t, map[string]float64{"": 3.00}, fareAmounts(t, c, journey(leg("rail1", "c", "d", at(8, 50), at(9, 10)))))
}

// joinFeed prices rail by zone. The station's platforms are in zone2.
var joinFeed = map[string]string{
	"areas.txt": `
area_id
zone1
zone2
zone3
zone4`,
	"stop_areas.txt": `
area_id,stop_id
zone1,a
zone2,station
zone3,d
zone4,b
zone4,c`,
	"fare_products.txt": `
fare_product_id,amount,currency
zone1_zone2,3.00,USD
zone2_zone3,3.00,USD
zone1_zone3,4.50,USD
zone1_zone4,2.00,USD
zone4_zone3,2.00,USD`,
	"fare_leg_rules.txt": `
network_id,from_area_id,to_area_id,fare_product_id
rail,zone1,zone2,zone1_zone2
rail,zone2,zone3,zone2_zone3
rail,zone1,zone3,zone1_zone3
rail,zone1,zone4,zone1_zone4
rail,zone4,zone3,zone4_zone3`,
}

func TestFareLegs_JoinRules(t *testing.T) {
	inStation := journey(
		leg("rail1", "a", "platform1", at(8, 0), at(8, 20)),
		leg("rail2", "platform2", "d", at(8, 30), at(8, 50)),
	)
	betweenStops := journey(
		leg("rail1", "a", "b", at(8, 0), at(8, 20)),
		leg("rail2", "c", "d", at(8, 30), at(8, 50)),
	)
	t.Run("without join rules", func(t *testing.T) {
		c := newTestCoster(t, joinFeed)
		assert.Equal(t, map[string]float64{"": 6.00}, fareAmounts(t, c, inStation))
	})
	t.Run("network join within a station", func(t *testing.T) {
		c := newTestCoster(t, joinFeed, map[string]string{
			"fare_leg_join_rules.txt": `
from_network_id,to_network_id
rail,rail`,
		})
		fare, err := c.LowestFare(inStation)
		require.NoError(t, err)
		require.NotNil(t, fare)
		assert.Equal(t, 4.50, fare.Amount)
		require.Len(t, fare.FareLegs, 1)
		assert.Equal(t, []int{0, 1}, fare.FareLegs[0].LegIndexes)
		// Without stops in the rule, a transfer between stations isn't joined.
		assert.Equal(t, map[string]float64{"": 4.00}, fareAmounts(t, c, betweenStops))
	})
	t.Run("join at stops", func(t *testing.T) {
		c := newTestCoster(t, joinFeed, map[string]string{
			"fare_leg_join_rules.txt": `
from_network_id,to_network_id,from_stop_id,to_stop_id
rail,rail,b,c`,
		})
		assert.Equal(t, map[string]float64{"": 4.50}, fareAmounts(t, c, betweenStops))
		assert.Equal(t, map[string]float64{"": 6.00}, fareAmounts(t, c, inStation))
	})
	t.Run("join at a station", func(t *testing.T) {
		// A station in the rule matches its platforms.
		c := newTestCoster(t, joinFeed, map[string]string{
			"fare_leg_join_rules.txt": `
from_network_id,to_network_id,from_stop_id,to_stop_id
rail,rail,station,station`,
		})
		assert.Equal(t, map[string]float64{"": 4.50}, fareAmounts(t, c, inStation))
	})
}
