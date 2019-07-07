package graph

import "github.com/interline-io/gotransit"

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
func BuildGraph(reader gotransit.Reader) (*EntityGraph, error) {
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
		if len(ent.AgencyID) == 0 {
			eg.AddEdge(dan, en)
		} else if agency, ok := eg.Node(NewNode("agency.txt", ent.AgencyID)); ok {
			eg.AddEdge(agency, en)
		}
	}
	// Add nodes for Calendars and Shapes
	for ent := range reader.Calendars() {
		eg.AddNode(entityNode(&ent))
	}
	for ent := range reader.CalendarDates() {
		eg.AddNode(NewNode("calendar.txt", ent.ServiceID))
	}
	for ent := range reader.Shapes() {
		eg.AddNode(entityNode(&ent))
	}
	// Add Trips and link
	for ent := range reader.Trips() {
		en, _ := eg.AddNode(entityNode(&ent))
		if r, ok := eg.Node(NewNode("routes.txt", ent.RouteID)); ok {
			eg.AddEdge(r, en)
		}
		if c, ok := eg.Node(NewNode("calendar.txt", ent.ServiceID)); ok {
			eg.AddEdge(c, en)
		}
		if len(ent.ShapeID) > 0 {
			if s, ok := eg.Node(NewNode("shapes.txt", ent.ShapeID)); ok {
				eg.AddEdge(s, en)
			}
		}
	}
	// Add Stops and link to parent stations
	ps := map[string]string{}   // parent stations
	fz := map[string][]string{} // farezones	1
	for ent := range reader.Stops() {
		en := entityNode(&ent)
		eg.AddNode(en)
		if ent.ParentStation.String != "" {
			ps[ent.StopID] = ent.ParentStation.String
		}
		if len(ent.ZoneID) > 0 {
			fz[ent.ZoneID] = append(fz[ent.ZoneID], ent.StopID)
		}
	}
	// Add stops to parent stops
	for k, sid := range ps {
		a, ok1 := eg.Node(NewNode("stops.txt", sid))
		b, ok2 := eg.Node(NewNode("stops.txt", k))
		if ok1 && ok2 {
			eg.AddEdge(a, b)
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
	//
	for ent := range reader.StopTimes() {
		t, _ := eg.Node(NewNode("trips.txt", ent.TripID))
		s, _ := eg.Node(NewNode("stops.txt", ent.StopID))
		eg.AddEdge(s, t)
	}
	// Add FareAttributes - FareRules will create child edges from Stops and Routes
	for ent := range reader.FareAttributes() {
		eg.AddNode(entityNode(&ent))
	}
	for ent := range reader.FareRules() {
		fn, _ := eg.Node(NewNode("fare_attributes.txt", ent.FareID))
		if zn, ok := eg.Node(NewNode("farezone", ent.OriginID)); ok {
			eg.AddEdge(fn, zn)
		}
		if zn, ok := eg.Node(NewNode("farezone", ent.DestinationID)); ok {
			eg.AddEdge(fn, zn)
		}
		if zn, ok := eg.Node(NewNode("farezone", ent.ContainsID)); ok {
			eg.AddEdge(fn, zn)
		}
	}
	return eg, nil
}
