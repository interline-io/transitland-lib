package coster

import (
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/adapters/direct"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchLegRules_EmptyFields(t *testing.T) {
	// Without rule_priority, an empty field matches only values that no rule lists.
	feed := baseFeed()
	feed.AreaList = areas("zone1", "zone2")
	feed.StopAreaList = []gtfs.StopArea{stopArea("zone1", "a"), stopArea("zone2", "b")}
	feed.FareProductList = []gtfs.FareProduct{
		fareProduct("bus_fare", 2.00, "", ""),
		fareProduct("other_fare", 1.00, "", ""),
		fareProduct("rail_fare", 4.00, "", ""),
		fareProduct("zone_fare", 5.00, "", ""),
	}
	feed.FareLegRuleList = []gtfs.FareLegRule{
		legRule("", "bus", "bus_fare"),
		legRule("", "", "other_fare"),
		areaRule("rail", "zone1", "zone2", "zone_fare"),
		legRule("", "rail", "rail_fare"),
	}
	c := newTestCoster(t, feed)
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
	feed := baseFeed()
	feed.AreaList = areas("zone_x", "zone_y", "zone_z")
	feed.StopAreaList = []gtfs.StopArea{stopArea("zone_x", "a"), stopArea("zone_z", "a"), stopArea("zone_y", "b")}
	feed.FareProductList = []gtfs.FareProduct{
		fareProduct("zone_fare", 5.00, "", ""),
		fareProduct("rail_fare", 4.00, "", ""),
	}
	feed.FareLegRuleList = []gtfs.FareLegRule{
		areaRule("rail", "zone_x", "zone_y", "zone_fare"),
		legRule("", "rail", "rail_fare"),
	}
	c := newTestCoster(t, feed)
	// No rule lists zone_z, so the empty rule matches the departure from a.
	assert.Equal(t, map[string]float64{"": 4.00}, fareAmounts(t, c, journey(leg("rail1", "a", "c", at(8, 0), at(8, 20)))))
	// A rule lists zone_y, so only the zone rule matches the arrival at b.
	assert.Equal(t, map[string]float64{"": 5.00}, fareAmounts(t, c, journey(leg("rail1", "a", "b", at(8, 0), at(8, 20)))))
}

func TestMatchLegRules_RulePriority(t *testing.T) {
	// With rule_priority, an empty field matches any value, and the highest priority wins.
	feed := baseFeed()
	feed.AreaList = areas("zone1", "zone2")
	feed.StopAreaList = []gtfs.StopArea{stopArea("zone1", "a"), stopArea("zone2", "b")}
	feed.FareProductList = []gtfs.FareProduct{
		fareProduct("flat_fare", 1.00, "", ""),
		fareProduct("rail_fare", 4.00, "", ""),
		fareProduct("zone_fare", 5.00, "", ""),
	}
	railRule := legRule("", "rail", "rail_fare")
	railRule.RulePriority = tt.NewInt(1)
	zoneRule := areaRule("rail", "zone1", "zone2", "zone_fare")
	zoneRule.RulePriority = tt.NewInt(2)
	feed.FareLegRuleList = []gtfs.FareLegRule{legRule("", "", "flat_fare"), railRule, zoneRule}
	c := newTestCoster(t, feed)
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
	feed := baseFeed()
	feed.AreaList = areas("zone1", "zone2", "zone3")
	feed.StopAreaList = []gtfs.StopArea{stopArea("zone1", "station"), stopArea("zone2", "platform2"), stopArea("zone3", "b")}
	feed.FareProductList = []gtfs.FareProduct{
		fareProduct("zone1_zone3", 3.00, "", ""),
		fareProduct("zone2_zone3", 3.50, "", ""),
	}
	feed.FareLegRuleList = []gtfs.FareLegRule{
		areaRule("rail", "zone1", "zone3", "zone1_zone3"),
		areaRule("rail", "zone2", "zone3", "zone2_zone3"),
	}
	c := newTestCoster(t, feed)
	assert.Equal(t, map[string]float64{"": 3.00}, fareAmounts(t, c, journey(leg("rail1", "platform1", "b", at(8, 0), at(8, 20)))))
	assert.Equal(t, map[string]float64{"": 3.50}, fareAmounts(t, c, journey(leg("rail1", "platform2", "b", at(8, 0), at(8, 20)))))
}

// timeframeFeed has peak and off-peak timeframes on weekdays and one all-day timeframe on weekends.
func timeframeFeed() *direct.Reader {
	feed := baseFeed()
	feed.CalendarList = []gtfs.Calendar{
		calendar("weekdays", time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday),
		calendar("weekends", time.Saturday, time.Sunday),
	}
	feed.TimeframeList = []gtfs.Timeframe{
		timeframe("peak", 7, 9, "weekdays"),
		timeframe("peak", 16, 18, "weekdays"),
		timeframe("offpeak", 0, 7, "weekdays"),
		timeframe("offpeak", 9, 16, "weekdays"),
		timeframe("offpeak", 18, 24, "weekdays"),
		// Without start and end times, a timeframe covers the whole day.
		{TimeframeGroupID: str("weekend"), ServiceID: key("weekends")},
	}
	feed.FareProductList = []gtfs.FareProduct{
		fareProduct("peak_fare", 5.00, "", ""),
		fareProduct("offpeak_fare", 3.00, "", ""),
		fareProduct("weekend_fare", 2.00, "", ""),
	}
	return feed
}

func TestMatchLegRules_Timeframes(t *testing.T) {
	feed := timeframeFeed()
	feed.FareLegRuleList = []gtfs.FareLegRule{
		{NetworkID: str("rail"), FromTimeframeGroupID: str("peak"), FareProductID: str("peak_fare")},
		{NetworkID: str("rail"), FromTimeframeGroupID: str("offpeak"), FareProductID: str("offpeak_fare")},
		{NetworkID: str("rail"), FromTimeframeGroupID: str("weekend"), FareProductID: str("weekend_fare")},
	}
	c := newTestCoster(t, feed)
	tcs := []struct {
		name string
		leg  Leg
		want float64
	}{
		{"weekday peak", leg("rail1", "c", "d", at(8, 0), at(8, 30)), 5.00},
		// 07:00 ends the first off-peak range and starts the peak.
		{"end_time is exclusive", leg("rail1", "c", "d", at(7, 0), at(7, 30)), 5.00},
		{"second peak range", leg("rail1", "c", "d", at(17, 59), at(18, 30)), 5.00},
		{"weekend", leg("rail1", "c", "d", oct(10, 8, 0), oct(10, 8, 30)), 2.00},
		// A Friday trip that departs after midnight departs on Saturday, the event's local date.
		{"after midnight", leg("rail1", "c", "d", oct(10, 0, 30), oct(10, 1, 0)), 2.00},
		// Friday 20:00 in California is already Saturday in UTC.
		{"local date", leg("rail1", "c", "d", oct(9, 20, 0), oct(9, 20, 30)), 3.00},
		// 08:00 in California is 11:00 at a stop in New York.
		{"stop timezone", leg("rail1", "east", "d", at(8, 0), at(8, 30)), 3.00},
		// A boarding area takes its station's timezone, two levels up.
		{"station timezone", leg("rail1", "east_boarding", "d", at(8, 0), at(8, 30)), 3.00},
		// A platform takes its station's timezone, not its own.
		{"platform timezone", leg("rail1", "platform3", "d", at(8, 0), at(8, 30)), 5.00},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, map[string]float64{"": tc.want}, fareAmounts(t, c, journey(tc.leg)))
		})
	}
}

func TestMatchLegRules_ToTimeframes(t *testing.T) {
	// to_timeframe_group_id matches the leg's arrival.
	feed := timeframeFeed()
	feed.FareLegRuleList = []gtfs.FareLegRule{
		{NetworkID: str("rail"), ToTimeframeGroupID: str("peak"), FareProductID: str("peak_fare")},
		{NetworkID: str("rail"), ToTimeframeGroupID: str("offpeak"), FareProductID: str("offpeak_fare")},
	}
	c := newTestCoster(t, feed)
	assert.Equal(t, map[string]float64{"": 5.00}, fareAmounts(t, c, journey(leg("rail1", "c", "d", at(6, 50), at(7, 10)))))
	assert.Equal(t, map[string]float64{"": 3.00}, fareAmounts(t, c, journey(leg("rail1", "c", "d", at(8, 50), at(9, 10)))))
}

// joinFeed prices rail by zone. The station's platforms are in zone2.
func joinFeed(joinRules ...gtfs.FareLegJoinRule) *direct.Reader {
	feed := baseFeed()
	feed.AreaList = areas("zone1", "zone2", "zone3", "zone4")
	feed.StopAreaList = []gtfs.StopArea{
		stopArea("zone1", "a"),
		stopArea("zone2", "station"),
		stopArea("zone3", "d"),
		stopArea("zone4", "b"),
		stopArea("zone4", "c"),
	}
	feed.FareProductList = []gtfs.FareProduct{
		fareProduct("zone1_zone2", 3.00, "", ""),
		fareProduct("zone2_zone3", 3.00, "", ""),
		fareProduct("zone1_zone3", 4.50, "", ""),
		fareProduct("zone1_zone4", 2.00, "", ""),
		fareProduct("zone4_zone3", 2.00, "", ""),
	}
	feed.FareLegRuleList = []gtfs.FareLegRule{
		areaRule("rail", "zone1", "zone2", "zone1_zone2"),
		areaRule("rail", "zone2", "zone3", "zone2_zone3"),
		areaRule("rail", "zone1", "zone3", "zone1_zone3"),
		areaRule("rail", "zone1", "zone4", "zone1_zone4"),
		areaRule("rail", "zone4", "zone3", "zone4_zone3"),
	}
	feed.FareLegJoinRuleList = joinRules
	return feed
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
		c := newTestCoster(t, joinFeed())
		assert.Equal(t, map[string]float64{"": 6.00}, fareAmounts(t, c, inStation))
	})
	t.Run("network join within a station", func(t *testing.T) {
		c := newTestCoster(t, joinFeed(gtfs.FareLegJoinRule{FromNetworkID: str("rail"), ToNetworkID: str("rail")}))
		fare, err := c.LowestFare(inStation)
		require.NoError(t, err)
		require.NotNil(t, fare)
		assert.Equal(t, 4.50, fare.Amount.InexactFloat64())
		require.Len(t, fare.FareLegs, 1)
		assert.Equal(t, []int{0, 1}, fare.FareLegs[0].LegIndexes)
		// Without stops in the rule, a transfer between stations isn't joined.
		assert.Equal(t, map[string]float64{"": 4.00}, fareAmounts(t, c, betweenStops))
	})
	t.Run("join at stops", func(t *testing.T) {
		c := newTestCoster(t, joinFeed(gtfs.FareLegJoinRule{FromNetworkID: str("rail"), ToNetworkID: str("rail"), FromStopID: str("b"), ToStopID: str("c")}))
		assert.Equal(t, map[string]float64{"": 4.50}, fareAmounts(t, c, betweenStops))
		assert.Equal(t, map[string]float64{"": 6.00}, fareAmounts(t, c, inStation))
	})
	t.Run("join at a station", func(t *testing.T) {
		// A station in the rule matches its platforms.
		c := newTestCoster(t, joinFeed(gtfs.FareLegJoinRule{FromNetworkID: str("rail"), ToNetworkID: str("rail"), FromStopID: str("station"), ToStopID: str("station")}))
		assert.Equal(t, map[string]float64{"": 4.50}, fareAmounts(t, c, inStation))
	})
}
