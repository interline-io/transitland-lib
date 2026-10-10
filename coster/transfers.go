package coster

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/shopspring/decimal"
)

// legOption is a leg rule matched to a fare leg, with the product row the rider would buy.
type legOption struct {
	rule        *gtfs.FareLegRule
	product     *gtfs.FareProduct
	amount      decimal.Decimal // the product row's amount
	fareMediaID string          // the product row's fare medium, empty when it names none
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
	amount   decimal.Decimal
	// State after the last fare leg
	fareMediaID string // fare medium of the last fare leg, empty when unknown
	transferred bool   // a transfer rule priced the transfer into the last fare leg
	runCount    int    // consecutive transfers within one leg group that end at the last fare leg
	runStart    int    // fare leg where those transfers began
}

// next returns the partial fare extended by one fare leg, reached by transfer t if not nil.
func (p partial) next(fl FareLeg, t *transferOption) partial {
	n := partial{
		fareLegs:    append(slices.Clip(p.fareLegs), fl),
		amount:      p.amount.Add(fl.Amount),
		fareMediaID: fl.FareMediaID,
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
				opt := legOption{rule: rule, product: p, amount: decimal.NewFromFloat(p.Amount.Val), fareMediaID: p.FareMediaID.Val}
				key := [2]string{rule.LegGroupID.Val, opt.fareMediaID}
				if j, ok := cheapest[key]; !ok {
					cheapest[key] = len(options[i])
					options[i] = append(options[i], opt)
				} else if opt.amount.LessThan(options[i][j].amount) {
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
			if best == nil || p.amount.LessThan(best.amount) {
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
				fl.Amount, fl.FareMediaID = opt.amount, opt.fareMediaID
				search(p.next(fl, nil))
				continue
			}
			// A fare leg whose product names no medium takes the transfer chain's medium.
			fl.FareMediaID = cmp.Or(opt.fareMediaID, p.fareMediaID)
			for _, t := range transfers {
				fl.TransferRule, fl.TransferProduct = t.rule, t.product
				fl.Amount = transferAmount(t, p, opt)
				search(p.next(fl, &t))
			}
		}
	}
	search(partial{})
	fare := &Fare{Amount: best.amount, Currency: best.fareLegs[0].Product.Currency.Val}
	for k, fl := range best.fareLegs {
		for _, p := range []*gtfs.FareProduct{fl.Product, fl.TransferProduct} {
			// tlib accepts currency codes in any case.
			if p != nil && !strings.EqualFold(p.Currency.Val, fare.Currency) {
				return nil, fmt.Errorf("fare mixes currencies %s and %s", fare.Currency, p.Currency.Val)
			}
		}
		// Give the caller copies of the Coster's entities, which other fares share.
		fl.LegRule, fl.Product = clone(fl.LegRule), clone(fl.Product)
		fl.TransferRule, fl.TransferProduct = clone(fl.TransferRule), clone(fl.TransferProduct)
		fl.HeldProductIDs = c.heldProductIDs(rules[k], r)
		fare.FareLegs = append(fare.FareLegs, fl)
	}
	return fare, nil
}

// clone returns a pointer to a copy of *v, or nil when v is nil.
func clone[T any](v *T) *T {
	if v == nil {
		return nil
	}
	ret := *v
	return &ret
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
	if p.fareMediaID != "" && next.fareMediaID != "" && p.fareMediaID != next.fareMediaID {
		// A transfer stays on the fare medium that paid for the previous fare leg. A product
		// that names no medium works with any medium.
		return nil
	}
	// The transfer's product must be sold on the fare legs' medium. When neither medium is
	// known, only a product row that names no medium applies.
	mediaID := cmp.Or(next.fareMediaID, p.fareMediaID)
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
		if id := rule.FilterFareProductID.Val; c.UseFilterFareProductID && id != "" && !r.held[id] {
			// The rule is only for riders who hold the filter's product.
			continue
		}
		t := transferOption{rule: rule}
		if id := rule.FareProductID.Val; id != "" {
			// A transfer whose product the rider can't buy with that medium doesn't apply.
			if t.product = c.product(id, r, mediaID); t.product == nil {
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
func transferAmount(t transferOption, p partial, next legOption) decimal.Decimal {
	ab := decimal.Zero
	if t.product != nil {
		ab = decimal.NewFromFloat(t.product.Amount.Val)
	}
	switch t.rule.FareTransferType.Val {
	case 1: // A + AB + B, then S + BC + C
		return ab.Add(next.amount)
	case 2: // AB, then S + BC
		if !p.transferred {
			// AB replaces A.
			return ab.Sub(p.fareLegs[len(p.fareLegs)-1].Amount)
		}
		return ab
	default: // A + AB, then S + BC
		return ab
	}
}
