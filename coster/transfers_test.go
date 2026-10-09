package coster

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// transferFeed puts each network in its own leg group, with products for transfer rules.
var transferFeed = map[string]string{
	"fare_products.txt": `
fare_product_id,amount,currency
bus_fare,2.00,USD
rail_fare,4.00,USD
ferry_fare,6.00,USD
ab,0.50,USD
bc,1.00,USD
discount,-1.00,USD
upgrade,3.00,USD`,
	"fare_leg_rules.txt": `
leg_group_id,network_id,fare_product_id
bus,bus,bus_fare
rail,rail,rail_fare
ferry,ferry,ferry_fare`,
}

func TestTransferOptions_FareTransferTypes(t *testing.T) {
	busToRail := journey(
		leg("bus1", "a", "b", at(8, 0), at(8, 20)),
		leg("rail1", "b", "c", at(8, 30), at(8, 50)),
	)
	tcs := []struct {
		name  string
		rules string
		want  float64
	}{
		{"no transfer rule", "", 6.00},
		{"A + AB", "bus,rail,0,ab", 2.50},
		{"A + AB + B", "bus,rail,1,discount", 5.00},
		{"AB", "bus,rail,2,upgrade", 3.00},
		{"rule without a product", "bus,rail,0,", 2.00},
		{"matching rule applies even when separate fares cost less", "bus,rail,1,ab", 6.50},
		{"rule in the other direction", "rail,bus,0,", 6.00},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestCoster(t, transferFeed, map[string]string{
				"fare_transfer_rules.txt": "from_leg_group_id,to_leg_group_id,fare_transfer_type,fare_product_id\n" + tc.rules,
			})
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
		busToRail   string
		railToFerry string
		want        float64
	}{
		{"0", "0", 2.00 + 0.50 + 1.00},
		{"1", "1", 2.00 + 0.50 + 4.00 + 1.00 + 6.00},
		{"2", "2", 0.50 + 1.00},
		{"0", "2", 2.00 + 0.50 + 1.00},
		{"2", "0", 0.50 + 1.00},
		{"2", "1", 0.50 + 1.00 + 6.00},
		{"1", "2", 2.00 + 0.50 + 4.00 + 1.00},
	}
	for _, tc := range tcs {
		t.Run(tc.busToRail+" then "+tc.railToFerry, func(t *testing.T) {
			c := newTestCoster(t, transferFeed, map[string]string{
				"fare_transfer_rules.txt": "from_leg_group_id,to_leg_group_id,fare_transfer_type,fare_product_id\n" +
					"bus,rail," + tc.busToRail + ",ab\n" +
					"rail,ferry," + tc.railToFerry + ",bc",
			})
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
		c := newTestCoster(t, transferFeed, map[string]string{
			"fare_transfer_rules.txt": `
from_leg_group_id,to_leg_group_id,transfer_count,fare_transfer_type,fare_product_id
bus,bus,1,0,
bus,bus,2,0,ab
bus,bus,-1,0,bc`,
		})
		assert.Equal(t, map[string]float64{"": 2.00}, fareAmounts(t, c, journey(buses[:2]...)))
		assert.Equal(t, map[string]float64{"": 2.00 + 0.50}, fareAmounts(t, c, journey(buses[:3]...)))
		assert.Equal(t, map[string]float64{"": 2.00 + 0.50 + 1.00}, fareAmounts(t, c, journey(buses...)))
	})
	t.Run("smallest transfer_count applies even when a larger one costs less", func(t *testing.T) {
		c := newTestCoster(t, transferFeed, map[string]string{
			"fare_transfer_rules.txt": `
from_leg_group_id,to_leg_group_id,transfer_count,fare_transfer_type,fare_product_id
bus,bus,1,0,ab
bus,bus,-1,0,`,
		})
		assert.Equal(t, map[string]float64{"": 2.00 + 0.50}, fareAmounts(t, c, journey(buses[:3]...)))
	})
	t.Run("transfers past transfer_count start a new fare", func(t *testing.T) {
		c := newTestCoster(t, transferFeed, map[string]string{
			"fare_transfer_rules.txt": `
from_leg_group_id,to_leg_group_id,transfer_count,fare_transfer_type,fare_product_id
bus,bus,1,0,`,
		})
		assert.Equal(t, map[string]float64{"": 4.00}, fareAmounts(t, c, journey(buses[:3]...)))
		assert.Equal(t, map[string]float64{"": 4.00}, fareAmounts(t, c, journey(buses...)))
	})
}

func TestTransferOptions_DurationLimit(t *testing.T) {
	// Bus 08:00-08:20, then rail 08:40-09:00. Measured by duration_limit_type, the transfer
	// takes 60 minutes (0), 40 (1), 20 (2), or 40 (3).
	busToRail := journey(
		leg("bus1", "a", "b", at(8, 0), at(8, 20)),
		leg("rail1", "b", "c", at(8, 40), at(9, 0)),
	)
	tcs := []struct {
		name  string
		limit string
		kind  string
		want  float64
	}{
		{"departure to arrival, over", "2700", "0", 6.00},
		{"departure to departure, within", "2700", "1", 2.00},
		{"departure to departure, over", "1800", "1", 6.00},
		{"arrival to departure, within", "1800", "2", 2.00},
		{"arrival to arrival, within", "2700", "3", 2.00},
		{"arrival to arrival, over", "1800", "3", 6.00},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestCoster(t, transferFeed, map[string]string{
				"fare_transfer_rules.txt": "from_leg_group_id,to_leg_group_id,duration_limit,duration_limit_type,fare_transfer_type\n" +
					"bus,rail," + tc.limit + "," + tc.kind + ",0",
			})
			assert.Equal(t, map[string]float64{"": tc.want}, fareAmounts(t, c, busToRail))
		})
	}
	t.Run("measured from the first leg of consecutive transfers", func(t *testing.T) {
		// Free transfers between buses for 90 minutes after the first departure.
		c := newTestCoster(t, transferFeed, map[string]string{
			"fare_transfer_rules.txt": `
from_leg_group_id,to_leg_group_id,transfer_count,duration_limit,duration_limit_type,fare_transfer_type
bus,bus,-1,5400,1,0`,
		})
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
	c := newTestCoster(t, transferFeed, map[string]string{
		"fare_transfer_rules.txt": `
from_leg_group_id,to_leg_group_id,fare_transfer_type,fare_product_id
bus,,0,
,ferry,1,discount`,
	})
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
	c := newTestCoster(t, transferFeed, map[string]string{
		"fare_leg_rules.txt": `
network_id,fare_product_id
bus,bus_fare`,
		"fare_transfer_rules.txt": `
from_leg_group_id,to_leg_group_id,transfer_count,fare_transfer_type
,,-1,0`,
	})
	j := journey(
		leg("bus1", "a", "b", at(8, 0), at(8, 20)),
		leg("bus2", "b", "c", at(8, 30), at(8, 50)),
	)
	assert.Equal(t, map[string]float64{"": 4.00}, fareAmounts(t, c, j))
}

func TestTransferOptions_FareMedia(t *testing.T) {
	// A transfer whose product the rider can't buy with their fare medium doesn't apply.
	c := newTestCoster(t, map[string]string{
		"fare_media.txt": `
fare_media_id,fare_media_name,fare_media_type
card,Card,2
cash,Cash,0`,
		"fare_products.txt": `
fare_product_id,amount,currency,fare_media_id
bus_fare,2.00,USD,card
bus_fare,2.50,USD,cash
card_transfer,0.00,USD,card`,
		"fare_leg_rules.txt": `
leg_group_id,network_id,fare_product_id
bus,bus,bus_fare`,
		"fare_transfer_rules.txt": `
from_leg_group_id,to_leg_group_id,transfer_count,fare_transfer_type,fare_product_id
bus,bus,-1,0,card_transfer`,
	})
	j := journey(
		leg("bus1", "a", "b", at(8, 0), at(8, 20)),
		leg("bus2", "b", "c", at(8, 30), at(8, 50)),
	)
	assert.Equal(t, map[string]float64{"card": 2.00, "cash": 5.00}, fareAmounts(t, c, j))
}
