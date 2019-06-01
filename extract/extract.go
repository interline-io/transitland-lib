package extract

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
		var agency *node
		if len(ent.AgencyID) == 0 {
			agency = dan
		} else {
			agency, _ = eg.Node(newNode("agency.txt", ent.AgencyID))
		}
		eg.AddNode(en)
		eg.AddEdge(agency, en)
	}
	//
	for ent := range reader.Calendars() {
		eg.AddNode(entityNode(&ent))
	}
	//
	for ent := range reader.CalendarDates() {
		a, _ := eg.Node(newNode("calendar.txt", ent.ServiceID))
		b := newNode("calendar_dates.txt", ent.ServiceID)
		eg.AddNode(b)
		eg.AddEdge(a, b)
	}
	//
	for ent := range reader.Shapes() {
		eg.AddNode(entityNode(&ent))
	}
	//
	for ent := range reader.Trips() {
		en := entityNode(&ent)
		eg.AddNode(en)
		r, _ := eg.Node(newNode("routes.txt", ent.RouteID))
		c, _ := eg.Node(newNode("calendar.txt", ent.ServiceID))
		eg.AddEdge(r, en)
		eg.AddEdge(en, c)
		if len(ent.ShapeID) > 0 {
			s, _ := eg.Node(newNode("shapes.txt", ent.ShapeID))
			eg.AddEdge(en, s)
		}
	}
	//
	ps := map[string]string{}
	for ent := range reader.Stops() {
		en := entityNode(&ent)
		eg.AddNode(en)
		if len(ent.ParentStation) > 0 {
			ps[ent.StopID] = ent.ParentStation
		}
	}
	for k, v := range ps {
		a, _ := eg.Node(newNode("stops.txt", v))
		b, _ := eg.Node(newNode("stops.txt", k))
		eg.AddEdge(a, b)
	}
	//
	for ent := range reader.StopTimes() {
		t, _ := eg.Node(newNode("trips.txt", ent.TripID))
		s, _ := eg.Node(newNode("stops.txt", ent.StopID))
		eg.AddEdge(t, s)
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
	//
	result := map[*node]bool{}
	em.graph.Search(foundNodes[:], false, func(n *node) {
		fmt.Println("child:", n)
		result[n] = true
	})
	em.graph.Search(foundNodes[:], true, func(n *node) {
		result[n] = true
	})
	em.found = result
}
