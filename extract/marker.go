package extract

import (
	"fmt"

	"github.com/interline-io/transitland-lib/internal/graph"
	"github.com/interline-io/transitland-lib/tl"
)

// TODO: Use found map[graph.Node] bool values, not pointers

// Marker selects entities specified during the Filter method.
type Marker struct {
	graph *graph.EntityGraph
	found map[*graph.Node]bool
}

// NewMarker returns a new Marker.
func NewMarker() Marker {
	return Marker{
		graph: graph.NewEntityGraph(),
		found: map[*graph.Node]bool{},
	}
}

// IsMarked returns if an Entity is marked.
func (em *Marker) IsMarked(filename, eid string) bool {
	if len(eid) == 0 {
		return true
	}
	if n, ok := em.graph.Node(graph.NewNode(filename, eid)); !ok {
		return false
	} else if _, ok2 := em.found[n]; ok2 {
		return true
	}
	return false
}

// IsVisited returns if an Entity was visited.
func (em *Marker) IsVisited(filename string, eid string) bool {
	_, ok := em.graph.Node(graph.NewNode(filename, eid))
	return ok
}

// Filter takes a Reader and selects any entities that are children of the specified file/id map.
func (em *Marker) Filter(reader tl.Reader, fm map[string][]string) error {
	eg, err := graph.BuildGraph(reader)
	if err != nil {
		return err
	}
	em.graph = eg
	foundNodes := []*graph.Node{}
	for k, v := range fm {
		for _, i := range v {
			if n, ok := em.graph.Node(graph.NewNode(k, i)); ok {
				foundNodes = append(foundNodes, n)
			} else {
				return fmt.Errorf("entity not found: %s '%s'", k, i)
			}
		}
	}
	// Find all children
	result := map[*graph.Node]bool{}
	em.graph.Search(foundNodes[:], false, func(n *graph.Node) {
		result[n] = true
	})
	// Now find parents of all found children
	check2 := []*graph.Node{}
	for k := range result {
		check2 = append(check2, k)
	}
	em.graph.Search(check2[:], true, func(n *graph.Node) {
		result[n] = true
	})
	em.found = result
	// log.Debugf("result: %#v\n", result)
	return nil
}
