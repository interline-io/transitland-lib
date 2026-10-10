package coster

import (
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/interline-io/transitland-lib/gtfs"
)

// legOption is a leg rule matched to a fare leg, with the product row the rider would use,
// what the rider pays for it, and the fare medium used to pay.
type legOption struct {
	rule        *gtfs.FareLegRule
	product     *gtfs.FareProduct
	amount      float64 // nothing when the rider holds the product
	fareMediaID string
}

// transferOption is a transfer rule that applies to a transfer, with the product row the
// rider would buy.
type transferOption struct {
	rule     *gtfs.FareTransferRule
	product  *gtfs.FareProduct // nil when the rule names no product, so the transfer is free
	runCount int               // transfers in the run within one leg group; 0 between groups
	runStart int               // fare leg where that run began
}

// partial is a fare priced through some of a journey's fare legs.
type partial struct {
	fareLegs []FareLeg
	amount   float64
	// State after the last fare leg
	fareMediaID string // fare medium of the last fare leg
	transferred bool   // a transfer rule priced the transfer into the last fare leg
	runCount    int    // consecutive transfers within one leg group that end at the last fare leg
	runStart    int    // fare leg where those transfers began
}

// next returns the partial fare extended by one fare leg, paid with a fare medium and reached
// by transfer t if not nil.
func (p partial) next(fl FareLeg, mediaID string, t *transferOption) partial {
	n := partial{
		fareLegs:    append(slices.Clip(p.fareLegs), fl),
		amount:      p.amount + fl.Amount,
		fareMediaID: mediaID,
	}
	if t != nil {
		n.transferred, n.runCount, n.runStart = true, t.runCount, t.runStart
	}
	return n
}

// lowestFare tries every way to price the fare legs for one rider, with any of the rider's
// fare media on each fare leg, and returns the lowest fare. It returns nil when some fare leg
// has no fare for the rider.
func (c *Coster) lowestFare(fareLegs []fareLeg, rules [][]*gtfs.FareLegRule, r rider) (*Fare, error) {
	options := make([][]legOption, len(fareLegs))
	for i := range fareLegs {
		// Transfers depend only on a fare leg's leg group and fare medium, so the cheapest
		// product for each pair prices the fare leg at least as low as any other.
		cheapest := map[[2]string]int{}
		for _, rule := range rules[i] {
			for _, mediaID := range r.fareMediaIDs {
				p := c.product(rule.FareProductID.Val, r, mediaID)
				if p == nil {
					continue
				}
				opt := legOption{rule: rule, product: p, amount: p.Amount.Val, fareMediaID: mediaID}
				if r.held[rule.FareProductID.Val] {
					opt.amount = 0
				}
				key := [2]string{rule.LegGroupID.Val, mediaID}
				if j, ok := cheapest[key]; !ok {
					cheapest[key] = len(options[i])
					options[i] = append(options[i], opt)
				} else if opt.amount < options[i][j].amount {
					options[i][j] = opt
				}
			}
		}
		if len(options[i]) == 0 {
			return nil, nil
		}
	}
	var best *partial
	var search func(p partial)
	search = func(p partial) {
		k := len(p.fareLegs)
		if k == len(fareLegs) {
			if best == nil || p.amount < best.amount {
				best = &p
			}
			return
		}
		for _, opt := range options[k] {
			fl := FareLeg{LegIndexes: fareLegs[k].indexes, LegRule: opt.rule, Product: opt.product}
			var transfers []transferOption
			if k > 0 {
				transfers = c.transferOptions(fareLegs, p, opt, r)
			}
			// A matching transfer rule must price the transfer. Without one, this fare leg
			// starts a new sub-journey and costs its own product.
			if len(transfers) == 0 {
				fl.Amount = opt.amount
				search(p.next(fl, opt.fareMediaID, nil))
				continue
			}
			for _, t := range transfers {
				fl.TransferRule, fl.TransferProduct = t.rule, t.product
				fl.Amount = transferAmount(t, p, opt)
				search(p.next(fl, opt.fareMediaID, &t))
			}
		}
	}
	search(partial{})
	fare := &Fare{
		Amount:   best.amount,
		Currency: best.fareLegs[0].Product.Currency.Val,
		FareLegs: best.fareLegs,
	}
	for _, fl := range fare.FareLegs {
		for _, p := range []*gtfs.FareProduct{fl.Product, fl.TransferProduct} {
			if p != nil && p.Currency.Val != fare.Currency {
				return nil, fmt.Errorf("fare mixes currencies %s and %s", fare.Currency, p.Currency.Val)
			}
		}
	}
	return fare, nil
}

// transferOptions returns the transfer rules that price the transfer from the partial
// fare's last fare leg into the next one, when it uses leg rule next.
func (c *Coster) transferOptions(fareLegs []fareLeg, p partial, next legOption, r rider) []transferOption {
	k := len(p.fareLegs)
	from, to := p.fareLegs[k-1].LegRule.LegGroupID.Val, next.rule.LegGroupID.Val
	if from == "" || to == "" {
		// Only legs in leg groups can transfer.
		return nil
	}
	if p.fareMediaID != next.fareMediaID {
		// A transfer stays on the fare medium that paid for the fare leg before it.
		return nil
	}
	// Consecutive transfers within one leg group form a run: transfer_count counts its
	// transfers, and duration_limit is measured from its first fare leg.
	count, start := 1, k-1
	if from == to && p.runCount > 0 {
		count, start = p.runCount+1, p.runStart
	}
	var ret []transferOption
	for i := range c.transferRules {
		rule := &c.transferRules[i]
		if !matchID(rule.FromLegGroupID.Val, from, c.listedFromGroups) ||
			!matchID(rule.ToLegGroupID.Val, to, c.listedToGroups) ||
			transferLimit(rule) < count ||
			!withinDuration(rule, fareLegs[start], fareLegs[k]) {
			continue
		}
		t := transferOption{rule: rule}
		if id := rule.FareProductID.Val; id != "" {
			// A transfer whose product the rider can't buy with that medium doesn't apply.
			if t.product = c.product(id, r, next.fareMediaID); t.product == nil {
				continue
			}
		}
		if from == to {
			t.runCount, t.runStart = count, start
		}
		ret = append(ret, t)
	}
	// When rules with different transfer_count values match, the spec selects the smallest.
	lowest := math.MaxInt
	for _, t := range ret {
		lowest = min(lowest, transferLimit(t.rule))
	}
	return slices.DeleteFunc(ret, func(t transferOption) bool { return transferLimit(t.rule) != lowest })
}

// transferLimit returns how many consecutive transfers a rule covers.
func transferLimit(rule *gtfs.FareTransferRule) int {
	if !rule.TransferCount.Valid || rule.TransferCount.Val < 0 {
		return math.MaxInt
	}
	return rule.TransferCount.Int()
}

// withinDuration reports whether a transfer sub-journey, from fare leg first through fare
// leg last, fits in the rule's duration_limit.
func withinDuration(rule *gtfs.FareTransferRule, first fareLeg, last fareLeg) bool {
	if !rule.DurationLimit.Valid {
		return true
	}
	var d time.Duration
	switch rule.DurationLimitType.Val {
	case 1: // departure to departure
		d = last.departure.Sub(first.departure)
	case 2: // arrival to departure
		d = last.departure.Sub(first.arrival)
	case 3: // arrival to arrival
		d = last.arrival.Sub(first.arrival)
	default: // departure to arrival
		d = last.arrival.Sub(first.departure)
	}
	return d <= time.Duration(rule.DurationLimit.Val)*time.Second
}

// transferAmount returns what a transfer adds to the fare, following the spec's cost
// processing for fare_transfer_type. The first transfer of a sub-journey builds on A, the
// preceding fare leg; later transfers add to S, the running total.
func transferAmount(t transferOption, p partial, next legOption) float64 {
	ab := 0.0
	if t.product != nil {
		ab = t.product.Amount.Val
	}
	switch t.rule.FareTransferType.Val {
	case 1: // A + AB + B, then S + BC + C
		return ab + next.amount
	case 2: // AB, then S + BC
		if !p.transferred {
			// AB replaces A.
			return ab - p.fareLegs[len(p.fareLegs)-1].Amount
		}
		return ab
	default: // A + AB, then S + BC
		return ab
	}
}
