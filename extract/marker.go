// Package extract provides tools and utilities for extracting subsets of GTFS feeds.
package extract

import (
	"fmt"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/internal/graph"
	"github.com/interline-io/transitland-lib/tlxy"
)

// TODO: Use found map[graph.Node] bool values, not pointers

// Marker selects entities specified during the Filter method.
type Marker struct {
	graph          *graph.EntityGraph
	found          map[*graph.Node]bool
	fm             map[string][]string
	ex             map[string][]string
	bbox           string
	defaultExclude bool
}

// NewMarker returns a new Marker.
func NewMarker() Marker {
	return Marker{
		graph: graph.NewEntityGraph(),
		found: map[*graph.Node]bool{},
		fm:    map[string][]string{},
		ex:    map[string][]string{},
	}
}

func (em *Marker) SetBbox(bbox string) error {
	_, err := tlxy.ParseBbox(bbox)
	if err != nil {
		return err
	}
	em.bbox = bbox
	return nil
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

func (em *Marker) AddExclude(filename string, eid string) {
	em.ex[filename] = append(em.ex[filename], eid)
}

func (em *Marker) AddInclude(filename string, eid string) {
	em.fm[filename] = append(em.fm[filename], eid)
}

func (em *Marker) Count() int {
	c := 0
	if em.bbox != "" {
		c += 1
	}
	for _, v := range em.fm {
		c += len(v)
	}
	for _, v := range em.ex {
		c += len(v)
	}
	return c
}

// Filter takes a Reader and selects any entities that are children of the specified file/id map.
func (em *Marker) Filter(reader adapters.Reader) error {
	var bboxExcludeStops []string
	if em.bbox != "" {
		bbox, err := tlxy.ParseBbox(em.bbox)
		if err != nil {
			return err
		}
		for stop := range reader.Stops() {
			spt := tlxy.Point{
				Lon: stop.Geometry.X(),
				Lat: stop.Geometry.Y(),
			}
			if bbox.Contains(spt) {
				em.AddInclude("stops.txt", stop.StopID.Val)
			} else {
				bboxExcludeStops = append(bboxExcludeStops, stop.StopID.Val)
			}
		}
	}

	eg, err := graph.BuildGraph(reader)
	if err != nil {
		return err
	}
	em.graph = eg
	foundNodes := []*graph.Node{}
	for k, v := range em.fm {
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
	for k, v := range em.ex {
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

	// Exclude any stops outside of provided bbox
	for _, sid := range bboxExcludeStops {
		em.Mark("stops.txt", sid, false)
	}

	// log.Debugf("result: %#v\n", result)
	return nil
}
