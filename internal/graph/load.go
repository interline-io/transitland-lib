package graph

import "github.com/interline-io/transitland-lib/adapters"

/*

Dependency relations between named entities.

agency
	route
		trip / stop_time
			stop

non-platform stops (inverted)
	station
		platform
			level
			pathway
			trip / stop_time

calendar / calendar_dates
	trip

shape
	trip

fare_attribute / fare_rule (inverted)
	farezone (virtual)
		platform

-------

fare_rule: route present and marked, or at least 1 hit in origin/destination/contains
feed_info: always included

*/

// we just need EntityID / Filename
type entity interface {
	EntityID() string
	Filename() string
}

// entityNode convenience method
func entityNode(ent entity) *Node {
	return &Node{
		Filename: ent.Filename(),
		ID:       ent.EntityID(),
	}
}

// BuildGraph .
func BuildGraph(reader adapters.Reader) (*EntityGraph, error) {
	eg := NewEntityGraph()

	// Add Agencies and select default Agency
	var dan *Node
	for ent := range reader.Agencies() {
		en := entityNode(&ent)
		eg.AddNode(en)
		dan = en
	}

	// Add nodes for Routes and link to Agencies
	for ent := range reader.Routes() {
		en := entityNode(&ent)
		eg.AddNode(en)
		if !ent.AgencyID.Valid {
			eg.AddEdge(dan, en)
		} else if agency, ok := eg.Node(NewNode("agency.txt", ent.AgencyID.Val)); ok {
			eg.AddEdge(agency, en)
		}
	}

	// Add nodes for Calendars and Shapes
	for ent := range reader.Calendars() {
		eg.AddNode(entityNode(&ent))
	}
	for ent := range reader.CalendarDates() {
		eg.AddNode(NewNode("calendar.txt", ent.ServiceID.Val))
	}
	for ent := range reader.Shapes() {
		eg.AddNode(entityNode(&ent))
	}

	// Add Trips and link
	for ent := range reader.Trips() {
		en, _ := eg.AddNode(entityNode(&ent))
		if r, ok := eg.Node(NewNode("routes.txt", ent.RouteID.Val)); ok {
			eg.AddEdge(r, en)
		}
		if c, ok := eg.Node(NewNode("calendar.txt", ent.ServiceID.Val)); ok {
			eg.AddEdge(c, en)
		}
		if ent.ShapeID.Valid {
			if s, ok := eg.Node(NewNode("shapes.txt", ent.ShapeID.Val)); ok {
				eg.AddEdge(s, en)
			}
		}
	}

	// Add nodes for Levels
	for ent := range reader.Levels() {
		eg.AddNode(entityNode(&ent))
	}

	// Add Stops and link to parent stations
	ps := map[string]string{}   // parent stations
	cs := map[string][]string{} // non-platform stops in stations
	fz := map[string][]string{} // farezones	1
	for ent := range reader.Stops() {
		en := entityNode(&ent)
		eg.AddNode(en)
		if ent.ParentStation.Valid {
			ps[ent.StopID.Val] = ent.ParentStation.Val
			cs[ent.ParentStation.Val] = append(cs[ent.ParentStation.Val], ent.StopID.Val)
		}
		if ent.ZoneID.Valid {
			fz[ent.ZoneID.Val] = append(fz[ent.ZoneID.Val], ent.StopID.Val)
		}
		// Link levels
		if ent.LevelID.Valid {
			ln, _ := eg.Node(NewNode("levels.txt", ent.LevelID.Val))
			eg.AddEdge(ln, en)
		}
	}

	// Add stops to parent stops
	for sid, parentid := range ps {
		a, ok1 := eg.Node(NewNode("stops.txt", parentid))
		b, ok2 := eg.Node(NewNode("stops.txt", sid))
		if ok1 && ok2 {
			eg.AddEdge(a, b)
		}
		// Add non-platform stops, inverted as parents of station
		for _, npsid := range cs[parentid] {
			c, ok3 := eg.Node(NewNode("stops.txt", npsid))
			if ok1 && ok3 {
				eg.AddEdge(c, a)
			}
		}
	}

	// Add stops to farezones
	for k, sids := range fz {
		fn, _ := eg.AddNode(NewNode("farezone", k))
		for _, sid := range sids {
			if sn, ok := eg.Node(NewNode("stops.txt", sid)); ok {
				eg.AddEdge(fn, sn)
			}
		}
	}

	// Add pathways and link to stops
	for ent := range reader.Pathways() {
		pn, _ := eg.AddNode(entityNode(&ent))
		if fn, ok := eg.Node(NewNode("stops.txt", ent.FromStopID.Val)); ok {
			eg.AddEdge(fn, pn)
			eg.AddEdge(pn, fn)
		}
		if tn, ok := eg.Node(NewNode("stops.txt", ent.ToStopID.Val)); ok {
			eg.AddEdge(tn, pn)
			eg.AddEdge(pn, tn)
		}
	}

	// Stop Times
	for ent := range reader.StopTimes() {
		// Skip flex stop_times that reference locations instead of stops
		if !ent.StopID.Valid {
			continue
		}
		t, _ := eg.Node(NewNode("trips.txt", ent.TripID.Val))
		s, _ := eg.Node(NewNode("stops.txt", ent.StopID.Val))
		eg.AddEdge(s, t)
	}

	// Add FareAttributes - FareRules will create child edges from Stops and Routes
	for ent := range reader.FareAttributes() {
		eg.AddNode(entityNode(&ent))
	}
	for ent := range reader.FareRules() {
		fn, _ := eg.Node(NewNode("fare_attributes.txt", ent.FareID.Val))
		if zn, ok := eg.Node(NewNode("farezone", ent.OriginID.Val)); ok {
			eg.AddEdge(fn, zn)
		}
		if zn, ok := eg.Node(NewNode("farezone", ent.DestinationID.Val)); ok {
			eg.AddEdge(fn, zn)
		}
		if zn, ok := eg.Node(NewNode("farezone", ent.ContainsID.Val)); ok {
			eg.AddEdge(fn, zn)
		}
	}

	return eg, nil
}
