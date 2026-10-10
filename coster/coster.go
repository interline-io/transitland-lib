// Package coster calculates journey fares from GTFS Fares v2 data.
//
// It follows the adopted specification, including effective fare legs from
// fare_leg_join_rules.txt, rule_priority, timeframes, and chained transfers. It ignores
// extensions outside the adopted spec, such as transfer_only, except for two: fare products
// with the draft duration fields are passes, which never price a trip, and a Coster can
// honor filter_fare_product_id. It is based on an earlier internal implementation.
package coster

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/service"
	"github.com/shopspring/decimal"
)

// Journey is a rider's transit legs, in travel order.
type Journey struct {
	Legs []Leg
	// RiderCategoryIDs lists the categories that apply to the rider, by rider_category_id.
	// When empty, the feed's default rider categories apply.
	RiderCategoryIDs []string
	// FareProductIDs lists products the rider already holds, such as passes. Each fare leg
	// lists the held products that may cover the leg, and the leg still costs its full fare.
	FareProductIDs []string
	// FareMediaIDs lists the fare media available to the rider.
	// When empty, every fare medium in the feed is a candidate.
	FareMediaIDs []string
}

// Leg is one transit leg of a journey. Its route and stop IDs, and the journey's fare media
// IDs, use the same IDs as the Coster's source: each entity's own ID from New, or the
// writer's IDs from a Builder.
type Leg struct {
	RouteID       string
	FromStopID    string
	ToStopID      string
	DepartureTime time.Time
	ArrivalTime   time.Time
}

// Fare is the lowest fare for a journey.
type Fare struct {
	Amount   decimal.Decimal
	Currency string
	FareLegs []FareLeg
}

// FareLeg is one effective fare leg of a fare and the amount it adds to the fare.
type FareLeg struct {
	LegIndexes      []int // journey legs joined into this fare leg
	LegRule         *gtfs.FareLegRule
	Product         *gtfs.FareProduct      // the row of the leg rule's product that the rider buys
	TransferRule    *gtfs.FareTransferRule // rule for the transfer into this fare leg, if any
	TransferProduct *gtfs.FareProduct      // the transfer rule's product, if it names one
	Amount          decimal.Decimal
	// FareMediaID is the fare medium used to pay. A fare leg whose product names no medium
	// takes the medium of the transfer chain it joins, and is empty when that's unknown.
	FareMediaID string
	// HeldProductIDs lists products the rider holds, such as passes, that may cover this
	// fare leg. Amount is still the full fare.
	HeldProductIDs []string
}

// Coster calculates fares for journeys on one feed.
type Coster struct {
	// UseFilterFareProductID honors the proposed filter_fare_product_id extension: a transfer
	// rule with a filter applies only to riders who hold the filter's fare product.
	UseFilterFareProductID bool

	stops           map[string]gtfs.Stop
	stopAreas       map[string][]string
	routeNetworks   map[string]string // network_id by route_id, empty for routes without one
	agencyTimezone  string
	locations       map[string]*time.Location
	services        map[string]*service.Service
	timeframes      map[string][]gtfs.Timeframe
	products        map[string][]gtfs.FareProduct
	fareMediaIDs    []string
	riderCategories map[string]gtfs.RiderCategory
	legRules        []gtfs.FareLegRule
	joinRules       []gtfs.FareLegJoinRule
	transferRules   []gtfs.FareTransferRule
	// The spec bases empty-field matching on whether the rule_priority column exists.
	// Parsed entities don't show columns, so any rule that sets rule_priority counts.
	rulePriority bool
	// Values each column lists, for matching empty fields
	listedNetworks   map[string]bool
	listedFromAreas  map[string]bool
	listedToAreas    map[string]bool
	listedFromGroups map[string]bool
	listedToGroups   map[string]bool
}

// LowestFare returns the lowest fare for a journey. Each fare leg can use any of the rider's
// fare media, but a transfer stays on one medium. It returns nil when some fare leg has no
// fare for the rider.
func (c *Coster) LowestFare(journey Journey) (*Fare, error) {
	categories, err := c.riderCategorySet(journey.RiderCategoryIDs)
	if err != nil {
		return nil, err
	}
	held, err := c.heldProducts(journey.FareProductIDs)
	if err != nil {
		return nil, err
	}
	mediaIDs, err := c.fareMediaOptions(journey.FareMediaIDs)
	if err != nil {
		return nil, err
	}
	fareLegs, err := c.fareLegs(journey.Legs)
	if err != nil {
		return nil, err
	}
	rules := make([][]*gtfs.FareLegRule, len(fareLegs))
	for i, fl := range fareLegs {
		if rules[i], err = c.matchLegRules(fl); err != nil {
			return nil, err
		}
	}
	return c.lowestFare(fareLegs, rules, rider{categories: categories, held: held, fareMediaIDs: mediaIDs})
}

// rider describes who pays a fare: their rider categories, the products they hold, and the
// fare media they can use.
type rider struct {
	categories   map[string]bool
	held         map[string]bool
	fareMediaIDs []string
}

// riderCategorySet returns the rider categories for pricing a journey.
func (c *Coster) riderCategorySet(ids []string) (map[string]bool, error) {
	ret := map[string]bool{}
	for _, id := range ids {
		if _, ok := c.riderCategories[id]; !ok {
			return nil, fmt.Errorf("unknown rider category %q", id)
		}
		ret[id] = true
	}
	if len(ids) == 0 {
		for id, rc := range c.riderCategories {
			if rc.IsDefaultFareCategory.Val == 1 {
				ret[id] = true
			}
		}
	}
	return ret, nil
}

// heldProducts returns the fare products a rider holds.
func (c *Coster) heldProducts(ids []string) (map[string]bool, error) {
	ret := map[string]bool{}
	for _, id := range ids {
		if _, ok := c.products[id]; !ok {
			return nil, fmt.Errorf("unknown fare product %q", id)
		}
		ret[id] = true
	}
	return ret, nil
}

// heldProductIDs returns the products that a fare leg's rules name and the rider holds for
// one of the rider's categories.
func (c *Coster) heldProductIDs(rules []*gtfs.FareLegRule, r rider) []string {
	var ret []string
	for _, rule := range rules {
		id := rule.FareProductID.Val
		if !r.held[id] || slices.Contains(ret, id) {
			continue
		}
		if slices.ContainsFunc(c.products[id], func(p gtfs.FareProduct) bool { return r.eligible(&p) }) {
			ret = append(ret, id)
		}
	}
	return ret
}

// eligible reports whether a product row is for one of the rider's categories. An empty
// rider_category_id doesn't restrict a row.
func (r rider) eligible(p *gtfs.FareProduct) bool {
	v := p.RiderCategoryID.Val
	return v == "" || r.categories[v]
}

// fareMediaOptions returns the fare media for pricing a journey. A feed without fare media
// has one empty option.
func (c *Coster) fareMediaOptions(ids []string) ([]string, error) {
	for _, id := range ids {
		if !slices.Contains(c.fareMediaIDs, id) {
			return nil, fmt.Errorf("unknown fare media %q", id)
		}
	}
	if len(ids) > 0 {
		return ids, nil
	}
	if len(c.fareMediaIDs) > 0 {
		return c.fareMediaIDs, nil
	}
	return []string{""}, nil
}

// product returns the cheapest row of a fare product that the rider can buy with a fare
// medium, or nil when there is none. An empty rider_category_id or fare_media_id doesn't
// restrict a row.
func (c *Coster) product(id string, r rider, mediaID string) *gtfs.FareProduct {
	var best *gtfs.FareProduct
	rows := c.products[id]
	for i := range rows {
		p := &rows[i]
		if p.DurationAmount.Valid {
			// A row with the draft duration fields is a pass, which never prices a trip.
			continue
		}
		if !r.eligible(p) {
			continue
		}
		if v := p.FareMediaID.Val; v != "" && v != mediaID {
			continue
		}
		if best == nil || p.Amount.Val < best.Amount.Val {
			best = p
		}
	}
	return best
}

// location returns the timezone of fare events at a stop: the stop_timezone of its station,
// or of the stop itself when it has no parent, or else the feed's agency timezone.
func (c *Coster) location(stopID string) (*time.Location, error) {
	name := c.agencyTimezone
	// The spec gives a stop in a station the station's timezone instead of its own.
	if tz := c.stops[c.station(stopID)].StopTimezone.Val; tz != "" {
		name = tz
	}
	loc, ok := c.locations[name]
	if !ok {
		return nil, errors.New("feed has no agency timezone")
	}
	return loc, nil
}
