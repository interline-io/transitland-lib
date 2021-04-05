package graph

type Node struct {
	Filename string
	ID       string
}

func NewNode(filename, id string) *Node {
	return &Node{
		Filename: filename,
		ID:       id,
	}
}

type EntityGraph struct {
	Nodes    map[Node]*Node
	Parents  map[Node][]*Node
	Children map[Node][]*Node
}

func NewEntityGraph() *EntityGraph {
	return &EntityGraph{
		Nodes:    map[Node]*Node{},
		Parents:  map[Node][]*Node{},
		Children: map[Node][]*Node{},
	}
}

func (eg *EntityGraph) Node(n *Node) (*Node, bool) {
	found, ok := eg.Nodes[*n]
	return found, ok
}

func (eg *EntityGraph) AddNode(n *Node) (*Node, bool) {
	if found, ok := eg.Nodes[*n]; ok {
		return found, false
	}
	eg.Nodes[*n] = n
	return n, true
}

func (eg *EntityGraph) AddEdge(n1, n2 *Node) bool {
	eg.Children[*n1] = append(eg.Children[*n1], n2) // deref
	eg.Parents[*n2] = append(eg.Parents[*n2], n1)   // deref
	return true
}

func (eg *EntityGraph) findChildren(n *Node) ([]*Node, bool) {
	e, ok := eg.Children[*n]
	return e, ok
}

func (eg *EntityGraph) findParents(n *Node) ([]*Node, bool) {
	e, ok := eg.Parents[*n]
	return e, ok
}

func (eg *EntityGraph) Search(queue []*Node, up bool, f func(*Node)) {
	visited := map[*Node]bool{}
	for {
		if len(queue) == 0 {
			break
		}
		cur := queue[0]
		f(cur)
		queue = queue[1:]
		var edges []*Node
		if up == false {
			edges, _ = eg.findChildren(cur)
		} else {
			edges, _ = eg.findParents(cur)
		}
		for i := 0; i < len(edges); i++ {
			j := edges[i]
			if !visited[j] {
				queue = append(queue, j)
				visited[j] = true
			}
		}
	}
}
