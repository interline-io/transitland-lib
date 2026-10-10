package coster

import (
	"fmt"
	"testing"

	"github.com/interline-io/transitland-lib/adapters/direct"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// transferFeed puts each network in its own leg group, with products for transfer rules.
func transferFeed(rules ...gtfs.FareTransferRule) *direct.Reader {
	feed := baseFeed()
	feed.FareProductList = []gtfs.FareProduct{
		fareProduct("bus_fare", 2.00, "", ""),
		fareProduct("rail_fare", 4.00, "", ""),
		fareProduct("ferry_fare", 6.00, "", ""),
		fareProduct("ab", 0.50, "", ""),
		fareProduct("bc", 1.00, "", ""),
		fareProduct("discount", -1.00, "", ""),
		fareProduct("upgrade", 3.00, "", ""),
	}
	feed.FareLegRuleList = []gtfs.FareLegRule{
		legRule("bus", "bus", "bus_fare"),
		legRule("rail", "rail", "rail_fare"),
		legRule("ferry", "ferry", "ferry_fare"),
	}
	feed.FareTransferRuleList = rules
	return feed
}

// countRule returns a transfer rule that covers count transfers within a leg group, or any
// number of transfers when count is -1.
func countRule(group string, count int, transferType int, product string) gtfs.FareTransferRule {
	ret := transferRule(group, group, transferType, product)
	ret.TransferCount = tt.NewInt(count)
	return ret
}

func TestTransferOptions_FareTransferTypes(t *testing.T) {
	busToRail := journey(
		leg("bus1", "a", "b", at(8, 0), at(8, 20)),
		leg("rail1", "b", "c", at(8, 30), at(8, 50)),
	)
	tcs := []struct {
		name  string
		rules []gtfs.FareTransferRule
		want  float64
	}{
		{"no transfer rule", nil, 6.00},
		{"A + AB", []gtfs.FareTransferRule{transferRule("bus", "rail", 0, "ab")}, 2.50},
		{"A + AB + B", []gtfs.FareTransferRule{transferRule("bus", "rail", 1, "discount")}, 5.00},
		{"AB", []gtfs.FareTransferRule{transferRule("bus", "rail", 2, "upgrade")}, 3.00},
		{"rule without a product", []gtfs.FareTransferRule{transferRule("bus", "rail", 0, "")}, 2.00},
		{"matching rule applies even when separate fares cost less", []gtfs.FareTransferRule{transferRule("bus", "rail", 1, "ab")}, 6.50},
		{"rule in the other direction", []gtfs.FareTransferRule{transferRule("rail", "bus", 0, "")}, 6.00},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestCoster(t, transferFeed(tc.rules...))
			assert.Equal(t, map[string]float64{"": tc.want}, fareAmounts(t, c, busToRail))
		})
	}
}

func TestTransferOptions_ChainedTransfers(t *testing.T) {
	// The spec processes A > B as A + AB, A + AB + B, or AB, and then B > C on top of S,
	// the total so far: S + BC, S + BC + C, or S + BC.
	busRailFerry := journey(
		leg("bus1", "a", "b", at(8, 0), at(8, 20)),
		leg("rail1", "b", "c", at(8, 30), at(8, 50)),
		leg("ferry", "c", "d", at(9, 0), at(9, 30)),
	)
	tcs := []struct {
		busToRail   int
		railToFerry int
		want        float64
	}{
		{0, 0, 2.00 + 0.50 + 1.00},
		{1, 1, 2.00 + 0.50 + 4.00 + 1.00 + 6.00},
		{2, 2, 0.50 + 1.00},
		{0, 2, 2.00 + 0.50 + 1.00},
		{2, 0, 0.50 + 1.00},
		{2, 1, 0.50 + 1.00 + 6.00},
		{1, 2, 2.00 + 0.50 + 4.00 + 1.00},
	}
	for _, tc := range tcs {
		t.Run(fmt.Sprintf("%d then %d", tc.busToRail, tc.railToFerry), func(t *testing.T) {
			c := newTestCoster(t, transferFeed(
				transferRule("bus", "rail", tc.busToRail, "ab"),
				transferRule("rail", "ferry", tc.railToFerry, "bc"),
			))
			assert.Equal(t, map[string]float64{"": tc.want}, fareAmounts(t, c, busRailFerry))
		})
	}
}

func TestTransferOptions_TransferCount(t *testing.T) {
	buses := []Leg{
		leg("bus1", "a", "b", at(8, 0), at(8, 20)),
		leg("bus2", "b", "c", at(8, 30), at(8, 50)),
		leg("bus1", "c", "d", at(9, 0), at(9, 20)),
		leg("bus2", "d", "a", at(9, 30), at(9, 50)),
	}
	t.Run("smallest transfer_count that covers each transfer", func(t *testing.T) {
		c := newTestCoster(t, transferFeed(
			countRule("bus", 1, 0, ""),
			countRule("bus", 2, 0, "ab"),
			countRule("bus", -1, 0, "bc"),
		))
		assert.Equal(t, map[string]float64{"": 2.00}, fareAmounts(t, c, journey(buses[:2]...)))
		assert.Equal(t, map[string]float64{"": 2.00 + 0.50}, fareAmounts(t, c, journey(buses[:3]...)))
		assert.Equal(t, map[string]float64{"": 2.00 + 0.50 + 1.00}, fareAmounts(t, c, journey(buses...)))
	})
	t.Run("smallest transfer_count applies even when a larger one costs less", func(t *testing.T) {
		c := newTestCoster(t, transferFeed(countRule("bus", 1, 0, "ab"), countRule("bus", -1, 0, "")))
		assert.Equal(t, map[string]float64{"": 2.00 + 0.50}, fareAmounts(t, c, journey(buses[:3]...)))
	})
	t.Run("transfers past transfer_count start a new fare", func(t *testing.T) {
		c := newTestCoster(t, transferFeed(countRule("bus", 1, 0, "")))
		assert.Equal(t, map[string]float64{"": 4.00}, fareAmounts(t, c, journey(buses[:3]...)))
		assert.Equal(t, map[string]float64{"": 4.00}, fareAmounts(t, c, journey(buses...)))
	})
}

func TestTransferOptions_DurationLimit(t *testing.T) {
	// Bus 08:00-08:05, then rail 08:40-09:20. Measured by duration_limit_type, the transfer
	// takes 80 minutes (0), 40 (1), 35 (2), or 75 (3), so each limit below tells the types apart.
	busToRail := journey(
		leg("bus1", "a", "b", at(8, 0), at(8, 5)),
		leg("rail1", "b", "c", at(8, 40), at(9, 20)),
	)
	tcs := []struct {
		name      string
		limit     int
		limitType int
		want      float64
	}{
		{"departure to arrival, within", 4800, 0, 2.00},
		{"departure to arrival, over", 4500, 0, 6.00},
		{"departure to departure, within", 2700, 1, 2.00},
		{"departure to departure, over", 2220, 1, 6.00},
		{"arrival to departure, within", 2220, 2, 2.00},
		{"arrival to departure, over", 2040, 2, 6.00},
		{"arrival to arrival, within", 4500, 3, 2.00},
		{"arrival to arrival, over", 2700, 3, 6.00},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			rule := transferRule("bus", "rail", 0, "")
			rule.DurationLimit, rule.DurationLimitType = tt.NewInt(tc.limit), tt.NewInt(tc.limitType)
			c := newTestCoster(t, transferFeed(rule))
			assert.Equal(t, map[string]float64{"": tc.want}, fareAmounts(t, c, busToRail))
		})
	}
	t.Run("measured from the first leg of consecutive transfers", func(t *testing.T) {
		// Free transfers between buses for 90 minutes after the first departure.
		rule := countRule("bus", -1, 0, "")
		rule.DurationLimit, rule.DurationLimitType = tt.NewInt(5400), tt.NewInt(1)
		c := newTestCoster(t, transferFeed(rule))
		first := leg("bus1", "a", "b", at(8, 0), at(8, 10))
		second := leg("bus2", "b", "c", at(8, 40), at(8, 50))
		within := leg("bus1", "c", "d", at(9, 20), at(9, 30))
		// 100 minutes after the first departure, though only 60 after the second.
		over := leg("bus1", "c", "d", at(9, 40), at(9, 50))
		assert.Equal(t, map[string]float64{"": 2.00}, fareAmounts(t, c, journey(first, second, within)))
		assert.Equal(t, map[string]float64{"": 4.00}, fareAmounts(t, c, journey(first, second, over)))
	})
}

func TestTransferOptions_EmptyLegGroups(t *testing.T) {
	// An empty leg group matches the leg groups that its column doesn't list.
	c := newTestCoster(t, transferFeed(
		transferRule("bus", "", 0, ""),
		transferRule("", "ferry", 1, "discount"),
	))
	tcs := []struct {
		name  string
		first Leg
		next  Leg
		want  float64
	}{
		{"empty to_leg_group_id", leg("bus1", "a", "b", at(8, 0), at(8, 20)), leg("rail1", "b", "c", at(8, 30), at(8, 50)), 2.00},
		{"empty from_leg_group_id", leg("rail1", "a", "b", at(8, 0), at(8, 20)), leg("ferry", "b", "c", at(8, 30), at(8, 50)), 4.00 - 1.00 + 6.00},
		{"listed groups match neither rule", leg("bus1", "a", "b", at(8, 0), at(8, 20)), leg("ferry", "b", "c", at(8, 30), at(8, 50)), 8.00},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, map[string]float64{"": tc.want}, fareAmounts(t, c, journey(tc.first, tc.next)))
		})
	}
}

func TestTransferOptions_NoLegGroup(t *testing.T) {
	// A leg rule without a leg group can't transfer, even under a rule with empty leg groups.
	feed := transferFeed(transferRule("", "", 0, ""))
	feed.FareLegRuleList = []gtfs.FareLegRule{legRule("", "bus", "bus_fare")}
	c := newTestCoster(t, feed)
	j := journey(
		leg("bus1", "a", "b", at(8, 0), at(8, 20)),
		leg("bus2", "b", "c", at(8, 30), at(8, 50)),
	)
	assert.Equal(t, map[string]float64{"": 4.00}, fareAmounts(t, c, j))
}

func TestTransferOptions_FareMedia(t *testing.T) {
	// A transfer whose product the rider can't buy with their fare medium doesn't apply.
	feed := baseFeed()
	feed.FareMediaList = []gtfs.FareMedia{medium("card", 2), medium("cash", 0)}
	feed.FareProductList = []gtfs.FareProduct{
		fareProduct("bus_fare", 2.00, "", "card"),
		fareProduct("bus_fare", 2.50, "", "cash"),
		fareProduct("card_transfer", 0.00, "", "card"),
	}
	feed.FareLegRuleList = []gtfs.FareLegRule{legRule("bus", "bus", "bus_fare")}
	feed.FareTransferRuleList = []gtfs.FareTransferRule{countRule("bus", -1, 0, "card_transfer")}
	c := newTestCoster(t, feed)
	j := journey(
		leg("bus1", "a", "b", at(8, 0), at(8, 20)),
		leg("bus2", "b", "c", at(8, 30), at(8, 50)),
	)
	assert.Equal(t, map[string]float64{"card": 2.00, "cash": 5.00}, fareAmounts(t, c, j))
}

// mediaFeed sells rail only for cash and the ferry only for card, while the bus takes both.
func mediaFeed(rules ...gtfs.FareTransferRule) *direct.Reader {
	feed := baseFeed()
	feed.FareMediaList = []gtfs.FareMedia{medium("cash", 0), medium("card", 2)}
	feed.FareProductList = []gtfs.FareProduct{
		fareProduct("rail_fare", 4.00, "", "cash"),
		fareProduct("ferry_fare", 6.00, "", "card"),
		fareProduct("bus_fare", 2.00, "", "cash"),
		fareProduct("bus_fare", 2.50, "", "card"),
		fareProduct("free_transfer", 0.00, "", ""),
		fareProduct("card_transfer", 0.00, "", "card"),
	}
	feed.FareLegRuleList = []gtfs.FareLegRule{
		legRule("rail", "rail", "rail_fare"),
		legRule("ferry", "ferry", "ferry_fare"),
		legRule("bus", "bus", "bus_fare"),
	}
	feed.FareTransferRuleList = rules
	return feed
}

func TestTransferOptions_MixedMedia(t *testing.T) {
	t.Run("each fare leg uses a fare medium it sells, and no transfer crosses media", func(t *testing.T) {
		c := newTestCoster(t, mediaFeed(transferRule("rail", "ferry", 0, "free_transfer")))
		j := journey(
			leg("rail1", "a", "b", at(8, 0), at(8, 20)),
			leg("ferry", "b", "c", at(8, 30), at(8, 50)),
		)
		// No single fare medium prices both legs.
		assert.Equal(t, map[string]float64{}, fareAmounts(t, c, j))
		fare, err := c.LowestFare(j)
		require.NoError(t, err)
		require.NotNil(t, fare)
		assert.Equal(t, 4.00+6.00, fare.Amount.InexactFloat64())
		assert.Equal(t, "cash", fare.FareLegs[0].Product.FareMediaID.Val)
		assert.Equal(t, "card", fare.FareLegs[1].Product.FareMediaID.Val)
		assert.Equal(t, "cash", fare.FareLegs[0].FareMediaID)
		assert.Equal(t, "card", fare.FareLegs[1].FareMediaID)
		assert.Nil(t, fare.FareLegs[1].TransferRule)
	})
	t.Run("a transfer on one fare medium beats mixing media", func(t *testing.T) {
		c := newTestCoster(t, mediaFeed(countRule("bus", -1, 0, "card_transfer")))
		j := journey(
			leg("bus1", "a", "b", at(8, 0), at(8, 20)),
			leg("bus2", "b", "c", at(8, 30), at(8, 50)),
		)
		// Cash is cheaper for each ride, but only card gets the free transfer.
		fare, err := c.LowestFare(j)
		require.NoError(t, err)
		require.NotNil(t, fare)
		assert.Equal(t, 2.50, fare.Amount.InexactFloat64())
		for _, fl := range fare.FareLegs {
			assert.Equal(t, "card", fl.Product.FareMediaID.Val)
		}
	})
}

func TestTransferOptions_UnknownMedium(t *testing.T) {
	busToRail := journey(
		leg("bus1", "a", "b", at(8, 0), at(8, 20)),
		leg("rail1", "b", "c", at(8, 30), at(8, 50)),
	)
	t.Run("a product that names no medium never blocks a transfer", func(t *testing.T) {
		// Fare media that no product names don't let the legs split to skip the transfer.
		feed := transferFeed(transferRule("bus", "rail", 1, "ab"))
		feed.FareMediaList = []gtfs.FareMedia{medium("cash", 0), medium("card", 2)}
		c := newTestCoster(t, feed)
		assert.Equal(t, map[string]float64{"cash": 6.50, "card": 6.50}, fareAmounts(t, c, busToRail))
		fare, err := c.LowestFare(busToRail)
		require.NoError(t, err)
		require.NotNil(t, fare)
		assert.Equal(t, 6.50, fare.Amount.InexactFloat64())
		assert.NotNil(t, fare.FareLegs[1].TransferRule)
		assert.Equal(t, []string{"", ""}, []string{fare.FareLegs[0].FareMediaID, fare.FareLegs[1].FareMediaID})
	})
	t.Run("a fare leg whose product names no medium takes the transfer chain's medium", func(t *testing.T) {
		// Card-only ferry, a free card transfer to a bus that names no medium, then cash-only
		// rail with a free transfer from the bus. The bus continues the card chain, so the
		// cash rail can't.
		feed := baseFeed()
		feed.FareMediaList = []gtfs.FareMedia{medium("cash", 0), medium("card", 2)}
		feed.FareProductList = []gtfs.FareProduct{
			fareProduct("ferry_fare", 6.00, "", "card"),
			fareProduct("bus_fare", 2.00, "", ""),
			fareProduct("rail_fare", 4.00, "", "cash"),
			fareProduct("card_transfer", 0.00, "", "card"),
			fareProduct("free_transfer", 0.00, "", ""),
		}
		feed.FareLegRuleList = []gtfs.FareLegRule{
			legRule("ferry", "ferry", "ferry_fare"),
			legRule("bus", "bus", "bus_fare"),
			legRule("rail", "rail", "rail_fare"),
		}
		feed.FareTransferRuleList = []gtfs.FareTransferRule{
			transferRule("ferry", "bus", 0, "card_transfer"),
			transferRule("bus", "rail", 0, "free_transfer"),
		}
		fare, err := newTestCoster(t, feed).LowestFare(journey(
			leg("ferry", "a", "b", at(8, 0), at(8, 20)),
			leg("bus1", "b", "c", at(8, 30), at(8, 50)),
			leg("rail1", "c", "d", at(9, 0), at(9, 20)),
		))
		require.NoError(t, err)
		require.NotNil(t, fare)
		assert.Equal(t, 6.00+4.00, fare.Amount.InexactFloat64())
		assert.Equal(t, []string{"card", "card", "cash"}, []string{fare.FareLegs[0].FareMediaID, fare.FareLegs[1].FareMediaID, fare.FareLegs[2].FareMediaID})
		assert.NotNil(t, fare.FareLegs[1].TransferRule)
		assert.Nil(t, fare.FareLegs[2].TransferRule)
	})
	t.Run("a transfer product on one medium needs a fare leg on that medium", func(t *testing.T) {
		feed := transferFeed(transferRule("bus", "rail", 0, "card_transfer"))
		feed.FareMediaList = []gtfs.FareMedia{medium("cash", 0), medium("card", 2)}
		feed.FareProductList = append(feed.FareProductList, fareProduct("card_transfer", 0.00, "", "card"))
		assert.Equal(t, map[string]float64{"cash": 6.00, "card": 6.00}, fareAmounts(t, newTestCoster(t, feed), busToRail))
	})
}

func TestTransferOptions_FilterFareProductID(t *testing.T) {
	// Holders of the bus pass get a free transfer to rail.
	rule := transferRule("bus", "rail", 0, "")
	rule.FilterFareProductID = str("bus_pass")
	feed := baseFeed()
	feed.FareProductList = []gtfs.FareProduct{
		fareProduct("bus_fare", 2.00, "", ""),
		fareProduct("rail_fare", 4.00, "", ""),
		pass("bus_pass", 80.00, ""),
	}
	feed.FareLegRuleList = []gtfs.FareLegRule{legRule("bus", "bus", "bus_fare"), legRule("rail", "rail", "rail_fare")}
	feed.FareTransferRuleList = []gtfs.FareTransferRule{rule}
	busToRail := journey(
		leg("bus1", "a", "b", at(8, 0), at(8, 20)),
		leg("rail1", "b", "c", at(8, 30), at(8, 50)),
	)
	holder := busToRail
	holder.FareProductIDs = []string{"bus_pass"}
	t.Run("ignored by default", func(t *testing.T) {
		c := newTestCoster(t, feed)
		assert.Equal(t, map[string]float64{"": 2.00}, fareAmounts(t, c, busToRail))
	})
	t.Run("only for riders who hold the product", func(t *testing.T) {
		c := newTestCoster(t, feed)
		c.UseFilterFareProductID = true
		assert.Equal(t, map[string]float64{"": 6.00}, fareAmounts(t, c, busToRail))
		assert.Equal(t, map[string]float64{"": 2.00}, fareAmounts(t, c, holder))
	})
}
