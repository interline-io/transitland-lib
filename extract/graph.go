package extract 


type entity interface {
	EntityID() string
	Filename() string
}

type node struct {
	filename string
	id string
}

func newNode(filename, id string) *node {
	return &node{
		filename: filename,
		id: id,
	}
}

func entityNode(ent entity) *node {
	return &node{
		filename: ent.Filename(),
		id: ent.EntityID(),
	}
}

type entityGraph struct {
	nodes map[node]*node
	parents map[node][]*node
	children map[node][]*node
}

func newEntityGraph() *entityGraph {
	return &entityGraph{
		nodes: map[node]*node{},
		parents: map[node][]*node{},
		children: map[node][]*node{},
	}
}

func (eg *entityGraph) Node(n *node) (*node, bool) {
	found, ok := eg.nodes[*n]
	return found, ok
}

func (eg *entityGraph) AddNode(n *node) (*node, bool) {
	if found, ok := eg.nodes[*n]; ok {
		return found, false
	} 
	eg.nodes[*n] = n
	return n, true
}

func (eg *entityGraph) AddEdge(n1, n2 *node) bool {
	eg.children[*n1] = append(eg.children[*n1], n2) // deref
	eg.parents[*n2] = append(eg.parents[*n2], n1) // deref
	return true
}

func (eg *entityGraph) Children(n *node) ([]*node, bool) {
	e, ok := eg.children[*n]
	return e, ok
}

func (eg *entityGraph) Parents(n *node) ([]*node, bool) {
	e, ok := eg.parents[*n]
	return e, ok
}

func (eg *entityGraph) Search(queue []*node, up bool, f func(*node)) {
	visited := map[*node]bool{}
	for {
		if len(queue) == 0 {
			break
		}
		cur := queue[0]
		f(cur)
		queue = queue[1:len(queue)]
		var edges []*node
		if up == false {
			edges, _ = eg.Children(cur)
		} else {
			edges, _ = eg.Parents(cur)
		}
		for i:=0;i<len(edges);i++ {
			j := edges[i]
			if !visited[j] {
				queue = append(queue, j)
				visited[j] = true
			}
		}
	}
}
