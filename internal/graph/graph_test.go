package graph

import "testing"

func TestEntityGraph_AddNode(t *testing.T) {
	eg := NewEntityGraph()
	n := NewNode("", "1")
	eg.AddNode(n)
	if len(eg.Nodes) != 1 {
		t.Error("did not add node")
	}
}

func TestEntityGraph_AddEdge(t *testing.T) {
	eg := NewEntityGraph()
	n1 := NewNode("", "1")
	n2 := NewNode("", "2")
	eg.AddNode(n1)
	eg.AddNode(n2)
	eg.AddEdge(n1, n2)
	if e, ok := eg.findChildren(n1); !ok {
		t.Error("edge not found")
	} else if len(e) != 1 {
		t.Errorf("got %d expect %d edges", len(e), 1)
	}
}
