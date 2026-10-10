package coster

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/interline-io/transitland-lib/gtfs"
)

// fareLeg is an effective fare leg: one journey leg, or several that fare_leg_join_rules join.
type fareLeg struct {
	indexes   []int
	network   string // empty when the joined legs' networks differ
	fromStop  string
	toStop    string
	departure time.Time
	arrival   time.Time
}

// fareLegs checks a journey's legs and joins them into effective fare legs.
func (c *Coster) fareLegs(legs []Leg) ([]fareLeg, error) {
	if len(legs) == 0 {
		return nil, errors.New("journey has no legs")
	}
	var ret []fareLeg
	prevNetwork := ""
	for i, leg := range legs {
		network, ok := c.routeNetworks[leg.RouteID]
		if !ok {
			return nil, fmt.Errorf("leg %d: unknown route %q", i, leg.RouteID)
		}
		for _, id := range []string{leg.FromStopID, leg.ToStopID} {
			if _, ok := c.stops[id]; !ok {
				return nil, fmt.Errorf("leg %d: unknown stop %q", i, id)
			}
		}
		if leg.DepartureTime.IsZero() || leg.ArrivalTime.IsZero() {
			return nil, fmt.Errorf("leg %d: departure and arrival times are required", i)
		}
		if i > 0 && c.joins(legs[i-1], prevNetwork, leg, network) {
			fl := &ret[len(ret)-1]
			fl.indexes = append(fl.indexes, i)
			fl.toStop, fl.arrival = leg.ToStopID, leg.ArrivalTime
			if fl.network != network {
				fl.network = ""
			}
		} else {
			ret = append(ret, fareLeg{
				indexes:   []int{i},
				network:   network,
				fromStop:  leg.FromStopID,
				toStop:    leg.ToStopID,
				departure: leg.DepartureTime,
				arrival:   leg.ArrivalTime,
			})
		}
		prevNetwork = network
	}
	return ret, nil
}

// joins reports whether a fare_leg_join_rule joins the transfer from leg a to leg b.
// Blank fields don't restrict a rule.
func (c *Coster) joins(a Leg, aNetwork string, b Leg, bNetwork string) bool {
	for _, rule := range c.joinRules {
		if v := rule.FromNetworkID.Val; v != "" && v != aNetwork {
			continue
		}
		if v := rule.ToNetworkID.Val; v != "" && v != bNetwork {
			continue
		}
		from, to := rule.FromStopID.Val, rule.ToStopID.Val
		if from == "" && to == "" {
			// Without stops, the transfer must stay within one station.
			if c.station(a.ToStopID) != c.station(b.FromStopID) {
				continue
			}
		} else if (from != "" && !c.atStop(a.ToStopID, from)) || (to != "" && !c.atStop(b.FromStopID, to)) {
			continue
		}
		return true
	}
	return false
}

// matchLegRules returns the fare leg rules that match a fare leg.
func (c *Coster) matchLegRules(fl fareLeg) ([]*gtfs.FareLegRule, error) {
	// Without rule_priority, an empty field matches only values that no rule lists.
	// With it, an empty field matches any value.
	networks, fromAreas, toAreas := c.listedNetworks, c.listedFromAreas, c.listedToAreas
	if c.rulePriority {
		networks, fromAreas, toAreas = nil, nil, nil
	}
	legFromAreas, legToAreas := c.areas(fl.fromStop), c.areas(fl.toStop)
	var ret []*gtfs.FareLegRule
	for i := range c.legRules {
		rule := &c.legRules[i]
		if !matchID(rule.NetworkID.Val, fl.network, networks) ||
			!matchAreas(rule.FromAreaID.Val, legFromAreas, fromAreas) ||
			!matchAreas(rule.ToAreaID.Val, legToAreas, toAreas) {
			continue
		}
		fromOK, err := c.inTimeframe(rule.FromTimeframeGroupID.Val, fl.fromStop, fl.departure)
		if err != nil {
			return nil, err
		}
		toOK, err := c.inTimeframe(rule.ToTimeframeGroupID.Val, fl.toStop, fl.arrival)
		if err != nil {
			return nil, err
		}
		if fromOK && toOK {
			ret = append(ret, rule)
		}
	}
	if c.rulePriority {
		// Keep the rules with the highest rule_priority. An empty value counts as zero.
		top := int64(0)
		for _, rule := range ret {
			top = max(top, rule.RulePriority.Val)
		}
		ret = slices.DeleteFunc(ret, func(rule *gtfs.FareLegRule) bool { return rule.RulePriority.Val != top })
	}
	return ret, nil
}

// inTimeframe reports whether a fare event at a stop falls in a timeframe group. An empty
// group matches any time. The event's local date picks the active services, and its
// wall-clock time must fall in a timeframe's range.
func (c *Coster) inTimeframe(groupID string, stopID string, t time.Time) (bool, error) {
	if groupID == "" {
		return true, nil
	}
	loc, err := c.location(stopID)
	if err != nil {
		return false, err
	}
	local := t.In(loc)
	// Services compare dates at midnight UTC.
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
	secs := local.Hour()*3600 + local.Minute()*60 + local.Second()
	for _, tf := range c.timeframes[groupID] {
		// An empty start_time means 00:00:00, and an empty end_time means 24:00:00.
		start, end := 0, 24*3600
		if tf.StartTime.Valid {
			start = tf.StartTime.Int()
		}
		if tf.EndTime.Valid {
			end = tf.EndTime.Int()
		}
		if svc := c.services[tf.ServiceID.Val]; svc != nil && svc.IsActive(day) && secs >= start && secs < end {
			return true, nil
		}
	}
	return false, nil
}

// areas returns a stop's areas. A stop without areas of its own is in its parent station's.
func (c *Coster) areas(stopID string) []string {
	// GTFS nests stops at most three deep: boarding area, platform, station.
	for range 3 {
		if a := c.stopAreas[stopID]; len(a) > 0 {
			return a
		}
		if stopID = c.stops[stopID].ParentStation.Val; stopID == "" {
			break
		}
	}
	return nil
}

// station returns the station that contains a stop, or the stop itself when it has no parent.
func (c *Coster) station(stopID string) string {
	// A boarding area is two levels below its station.
	for range 2 {
		parent := c.stops[stopID].ParentStation.Val
		if parent == "" {
			break
		}
		stopID = parent
	}
	return stopID
}

// atStop reports whether a stop is target, or a child of target when target is a station.
func (c *Coster) atStop(stopID string, target string) bool {
	return stopID == target || c.station(stopID) == target
}

// matchID reports whether a rule field matches a value. An empty field matches any value
// not in listed.
func matchID(field string, value string, listed map[string]bool) bool {
	if field != "" {
		return field == value
	}
	return !listed[value]
}

// matchAreas reports whether a rule's area field matches a stop's areas. An empty field
// matches a stop without areas, or a stop with any area not in listed: the spec filters
// each of a stop's areas on its own.
func matchAreas(field string, areas []string, listed map[string]bool) bool {
	if field != "" {
		return slices.Contains(areas, field)
	}
	return len(areas) == 0 || slices.ContainsFunc(areas, func(a string) bool { return !listed[a] })
}
