package extract

import "testing"

func TestEntityGraph_AddNode(t *testing.T) {
	eg := newEntityGraph()
	n := newNode("", "1")
	eg.AddNode(n)
	if len(eg.nodes) != 1 {
		t.Error("did not add node")
	}
}

func TestEntityGraph_AddEdge(t *testing.T) {
	eg := newEntityGraph()
	n1 := newNode("", "1")
	n2 := newNode("", "2")
	eg.AddNode(n1)
	eg.AddNode(n2)
	eg.AddEdge(n1, n2)
	if e, ok := eg.Children(n1); !ok {
		t.Error("edge not found")
	} else if len(e) != 1 {
		t.Errorf("got %d expect %d edges", len(e), 1)
	}
}
