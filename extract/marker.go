package extract

import (
	"fmt"

	"github.com/interline-io/transitland-lib/internal/graph"
	"github.com/interline-io/transitland-lib/tl"
)

// TODO: Use found map[graph.Node] bool values, not pointers

// Marker selects entities specified during the Filter method.
type Marker struct {
	graph          *graph.EntityGraph
	found          map[*graph.Node]bool
	defaultExclude bool
}

// NewMarker returns a new Marker.
func NewMarker() Marker {
	return Marker{
		graph: graph.NewEntityGraph(),
		found: map[*graph.Node]bool{},
	}
}

func (em *Marker) Mark(filename string, eid string, val bool) {
	n, _ := em.graph.Node(graph.NewNode(filename, eid))
	em.found[n] = val
}

// IsMarked returns if an Entity is marked.
func (em *Marker) IsMarked(filename, eid string) bool {
	if len(eid) == 0 {
		return true
	}
	if n, ok := em.graph.Node(graph.NewNode(filename, eid)); !ok {
		return false
	} else if v, ok2 := em.found[n]; ok2 {
		return v
	}
	return !em.defaultExclude
}

// IsVisited returns if an Entity was visited.
func (em *Marker) IsVisited(filename string, eid string) bool {
	_, ok := em.graph.Node(graph.NewNode(filename, eid))
	return ok
}

// Filter takes a Reader and selects any entities that are children of the specified file/id map.
func (em *Marker) Filter(reader tl.Reader, fm map[string][]string, ex map[string][]string) error {
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
				return fmt.Errorf("included entity not found: %s '%s'", k, i)
			}
		}
	}
	// If any include options specified, default to exclude
	if len(foundNodes) > 0 {
		em.defaultExclude = true
	}

	var excludeNodes []*graph.Node
	for k, v := range ex {
		for _, i := range v {
			if n, ok := em.graph.Node(graph.NewNode(k, i)); ok {
				excludeNodes = append(excludeNodes, n)
			} else {
				return fmt.Errorf("excluded entity not found: %s '%s'", k, i)
			}
		}
	}

	// Find all children
	em.found = map[*graph.Node]bool{}
	em.graph.Search(foundNodes[:], false, func(n *graph.Node) {
		em.Mark(n.Filename, n.ID, true)
	})

	// Now find parents of all found children
	check2 := []*graph.Node{}
	for k := range em.found {
		check2 = append(check2, k)
	}
	em.graph.Search(check2[:], true, func(n *graph.Node) {
		em.Mark(n.Filename, n.ID, true)
	})

	// Now find children of all excluded nodes
	em.graph.Search(excludeNodes[:], false, func(n *graph.Node) {
		em.Mark(n.Filename, n.ID, false)
	})

	// log.Debugf("result: %#v\n", result)
	return nil
}
