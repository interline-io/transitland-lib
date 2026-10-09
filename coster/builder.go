package coster

import (
	"fmt"
	"time"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/service"
	"github.com/interline-io/transitland-lib/tt"
)

// New loads a Coster from a feed's fares, stops, routes, and calendars.
func New(reader adapters.Reader) (*Coster, error) {
	// Calendars come before calendar dates, and routes before route networks, as in a copy.
	b := NewBuilder()
	addAll(b, reader.Agencies())
	addAll(b, reader.Stops())
	addAll(b, reader.StopAreas())
	addAll(b, reader.Routes())
	addAll(b, reader.RouteNetworks())
	addAll(b, reader.Calendars())
	addAll(b, reader.CalendarDates())
	addAll(b, reader.Timeframes())
	addAll(b, reader.FareProducts())
	addAll(b, reader.FareMedia())
	addAll(b, reader.RiderCategories())
	addAll(b, reader.FareLegRules())
	addAll(b, reader.FareLegJoinRules())
	addAll(b, reader.FareTransferRules())
	return b.Coster()
}

// addAll adds each entity from a reader, keyed by its own ID.
func addAll[T any, PT interface {
	tt.Entity
	*T
}](b *Builder, ents chan T) {
	for ent := range ents {
		pt := PT(&ent)
		b.add(pt.EntityID(), pt)
	}
}

// Builder collects a Coster from the entities that a copy writes. Add it to the copy as
// a copier extension, and call Coster when the copy finishes.
type Builder struct {
	c *Coster
}

// NewBuilder returns an empty Builder.
func NewBuilder() *Builder {
	return &Builder{c: &Coster{
		stops:            map[string]gtfs.Stop{},
		stopAreas:        map[string][]string{},
		routeNetworks:    map[string]string{},
		locations:        map[string]*time.Location{},
		services:         map[string]*service.Service{},
		timeframes:       map[string][]gtfs.Timeframe{},
		products:         map[string][]gtfs.FareProduct{},
		riderCategories:  map[string]gtfs.RiderCategory{},
		listedNetworks:   map[string]bool{},
		listedFromAreas:  map[string]bool{},
		listedToAreas:    map[string]bool{},
		listedFromGroups: map[string]bool{},
		listedToGroups:   map[string]bool{},
	}}
}

// AfterWrite collects an entity from a copy. The copier rewrites references to the IDs
// that the writer returns, so the Builder keys entities by those IDs too.
func (b *Builder) AfterWrite(eid string, ent tt.Entity, emap *tt.EntityMap) error {
	b.add(eid, ent)
	return nil
}

// Coster loads the timezones that the collected agencies and stops name, and returns
// the Coster.
func (b *Builder) Coster() (*Coster, error) {
	c := b.c
	// Load each timezone once, so fare calculations only read shared state.
	names := []string{c.agencyTimezone}
	for _, stop := range c.stops {
		names = append(names, stop.StopTimezone.Val)
	}
	for _, name := range names {
		if _, ok := c.locations[name]; ok || name == "" {
			continue
		}
		loc, err := time.LoadLocation(name)
		if err != nil {
			return nil, fmt.Errorf("timezone %q: %w", name, err)
		}
		c.locations[name] = loc
	}
	return c, nil
}

// add collects one entity. Its references must use the same kind of ID as id.
func (b *Builder) add(id string, ent tt.Entity) {
	c := b.c
	switch v := ent.(type) {
	case *gtfs.Agency:
		if c.agencyTimezone == "" {
			c.agencyTimezone = v.AgencyTimezone.Val
		}
	case *gtfs.Stop:
		c.stops[id] = *v
	case *gtfs.StopArea:
		c.stopAreas[v.StopID.Val] = append(c.stopAreas[v.StopID.Val], v.AreaID.Val)
	case *gtfs.Route:
		c.routeNetworks[id] = v.NetworkID.Val
	case *gtfs.RouteNetwork:
		c.routeNetworks[v.RouteID.Val] = v.NetworkID.Val
	case *gtfs.Calendar:
		// Dates arrive separately, as CalendarDates.
		c.services[id] = service.NewService(*v)
	case *gtfs.CalendarDate:
		svc := c.services[v.ServiceID.Val]
		if svc == nil {
			// The service exists only in calendar_dates.txt.
			svc = service.NewService(gtfs.Calendar{ServiceID: tt.NewString(v.ServiceID.Val)})
			c.services[v.ServiceID.Val] = svc
		}
		svc.AddCalendarDate(*v)
	case *gtfs.Timeframe:
		c.timeframes[v.TimeframeGroupID.Val] = append(c.timeframes[v.TimeframeGroupID.Val], *v)
	case *gtfs.FareProduct:
		c.products[v.FareProductID.Val] = append(c.products[v.FareProductID.Val], *v)
	case *gtfs.FareMedia:
		c.fareMediaIDs = append(c.fareMediaIDs, id)
	case *gtfs.RiderCategory:
		c.riderCategories[v.RiderCategoryID.Val] = *v
	case *gtfs.FareLegRule:
		c.legRules = append(c.legRules, *v)
		c.rulePriority = c.rulePriority || v.RulePriority.Valid
		addListed(c.listedNetworks, v.NetworkID.Val)
		addListed(c.listedFromAreas, v.FromAreaID.Val)
		addListed(c.listedToAreas, v.ToAreaID.Val)
	case *gtfs.FareLegJoinRule:
		c.joinRules = append(c.joinRules, *v)
	case *gtfs.FareTransferRule:
		c.transferRules = append(c.transferRules, *v)
		addListed(c.listedFromGroups, v.FromLegGroupID.Val)
		addListed(c.listedToGroups, v.ToLegGroupID.Val)
	}
}

// addListed records a non-empty column value.
func addListed(listed map[string]bool, v string) {
	if v != "" {
		listed[v] = true
	}
}
