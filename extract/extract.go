package extract

/*

agency
	route
		trip / stop_time

station
	stop
		trip / stop_time

calendar / calendar_dates
	trip

shape
	trip

fare_attribute / fare_rule (inverted)
	farezone (virtual)
		stop

-------

fare_rule: route present and marked, or at least 1 hit in origin/destination/contains

*/

import (
	"fmt"

	"github.com/interline-io/gotransit"
)

// extractMarker is a Marker that visits all entities and marks those that are global or referenced by another Entity.
type extractMarker struct {
	graph *entityGraph
	found map[*node]bool
}

// NewExtractMarker returns a new extractMarker.
func NewExtractMarker() extractMarker {
	return extractMarker{
		graph: newEntityGraph(),
		found: map[*node]bool{},
	}
}

// IsMarked returns if an Entity is marked.
func (em *extractMarker) IsMarked(filename, eid string) bool {
	if len(eid) == 0 {
		return true
	}
	if n, ok := em.graph.Node(newNode(filename, eid)); !ok {
		return false
	} else if _, ok2 := em.found[n]; ok2 {
		return true
	}
	return false
}

// IsVisited returns if an Entity was visited.
func (em *extractMarker) IsVisited(filename, eid string) bool {
	_, ok := em.graph.Node(newNode(filename, eid))
	return ok
}

// Load .
func (em *extractMarker) Load(reader gotransit.Reader) error {
	eg := em.graph
	var dan *node
	for ent := range reader.Agencies() {
		en := entityNode(&ent)
		eg.AddNode(en)
		dan = en
	}
	//
	for ent := range reader.Routes() {
		en := entityNode(&ent)
		eg.AddNode(en)
		if len(ent.AgencyID) == 0 {
			eg.AddEdge(dan, en)
		} else if agency, ok := eg.Node(newNode("agency.txt", ent.AgencyID)); ok {
			eg.AddEdge(agency, en)
		}
	}
	//
	for ent := range reader.Calendars() {
		eg.AddNode(entityNode(&ent))
	}
	for ent := range reader.CalendarDates() {
		eg.AddNode(newNode("calendar.txt", ent.ServiceID))
	}
	//
	for ent := range reader.Shapes() {
		eg.AddNode(entityNode(&ent))
	}
	//
	for ent := range reader.Trips() {
		en, _ := eg.AddNode(entityNode(&ent))
		if r, ok := eg.Node(newNode("routes.txt", ent.RouteID)); ok {
			eg.AddEdge(r, en)
		}
		if c, ok := eg.Node(newNode("calendar.txt", ent.ServiceID)); ok {
			eg.AddEdge(c, en)
		}
		if len(ent.ShapeID) > 0 {
			if s, ok := eg.Node(newNode("shapes.txt", ent.ShapeID)); ok {
				eg.AddEdge(s, en)
			}
		}
	}
	//
	ps := map[string]string{}
	fz := map[string][]string{}
	for ent := range reader.Stops() {
		en := entityNode(&ent)
		eg.AddNode(en)
		if len(ent.ParentStation) > 0 {
			ps[ent.StopID] = ent.ParentStation
		}
		if len(ent.ZoneID) > 0 {
			fz[ent.ZoneID] = append(fz[ent.ZoneID], ent.StopID)
		}
	}
	// Add stops to parent stops
	for k, sid := range ps {
		a, ok1 := eg.Node(newNode("stops.txt", sid))
		b, ok2 := eg.Node(newNode("stops.txt", k))
		if ok1 && ok2 {
			eg.AddEdge(a, b)
		}
	}
	// Add stops to farezones
	for k, sids := range fz {
		fn, _ := eg.AddNode(newNode("farezone", k))
		for _, sid := range sids {
			if sn, ok := eg.Node(newNode("stops.txt", sid)); ok {
				eg.AddEdge(fn, sn)
			}
		}
	}
	//
	for ent := range reader.StopTimes() {
		t, _ := eg.Node(newNode("trips.txt", ent.TripID))
		s, _ := eg.Node(newNode("stops.txt", ent.StopID))
		eg.AddEdge(s, t)
	}
	// Add FareAttributes - FareRules will create child edges from Stops and Routes
	for ent := range reader.FareAttributes() {
		eg.AddNode(entityNode(&ent))
	}
	for ent := range reader.FareRules() {
		fn, _ := eg.Node(newNode("fare_attributes.txt", ent.FareID))
		if zn, ok := eg.Node(newNode("farezone", ent.OriginID)); ok {
			eg.AddEdge(fn, zn)
		}
		if zn, ok := eg.Node(newNode("farezone", ent.DestinationID)); ok {
			eg.AddEdge(fn, zn)
		}
		if zn, ok := eg.Node(newNode("farezone", ent.ContainsID)); ok {
			eg.AddEdge(fn, zn)
		}
	}
	return nil
}

func (em *extractMarker) Filter(fm map[string][]string) {
	foundNodes := []*node{}
	for k, v := range fm {
		for _, i := range v {
			if n, ok := em.graph.Node(newNode(k, i)); ok {
				foundNodes = append(foundNodes, n)
			}
		}
	}
	// Find all children
	result := map[*node]bool{}
	em.graph.Search(foundNodes[:], false, func(n *node) {
		fmt.Println("child:", n)
		result[n] = true
	})
	// Now find parents of all found children
	check2 := []*node{}
	for k := range result {
		check2 = append(check2, k)
	}
	em.graph.Search(check2[:], true, func(n *node) {
		result[n] = true
	})
	em.found = result
}
