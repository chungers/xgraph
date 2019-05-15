package xgraph // import "github.com/orkestr8/xgraph"

import (
	"sync"

	gonum "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

type node struct {
	Node
	id int64 // gonum id
}

func (n *node) ID() int64 {
	return n.id
}

type edge struct {
	from    Node
	to      Node
	kind    EdgeKind
	context []interface{}
}

func (e *edge) Kind() EdgeKind {
	return e.kind
}
func (e *edge) From() Node {
	return e.from
}
func (e *edge) To() Node {
	return e.to
}
func (e *edge) Context() []interface{} {
	return e.context
}

type graph struct {
	gonum.Builder
	Options

	directed map[EdgeKind]*directed
	nodeKeys map[interface{}]*node

	lock sync.RWMutex
}

func newGraph(options Options) *graph {
	return &graph{
		Builder:  simple.NewUndirectedGraph(),
		Options:  options,
		nodeKeys: map[interface{}]*node{},
		directed: map[EdgeKind]*directed{},
	}
}

func (g *graph) gonum(n Node, more ...Node) []gonum.Node {
	all := append([]Node{n}, more...)
	out := make([]gonum.Node, len(all))
	for i, xn := range all {
		if n, has := g.nodeKeys[xn.NodeKey()]; has {
			out[i] = n
		}
	}
	return out
}

/*
 Add registers the given Nodes to the graph.  Duplicate key but with different identity is not allowed.
*/
func (g *graph) Add(n Node, other ...Node) error {
	g.lock.Lock()
	defer g.lock.Unlock()

	all := append([]Node{n}, other...)
	for i := range all {
		found, has := g.nodeKeys[all[i].NodeKey()]
		if !has {
			newNode := &node{
				Node: all[i],
				id:   g.NewNode().ID(),
			}
			g.AddNode(newNode)
			g.nodeKeys[all[i].NodeKey()] = newNode
		} else if found.Node != all[i] {
			return ErrDuplicateKey{all[i]}
		}
	}

	return nil
}

func (g *graph) Node(k NodeKey) Node {
	g.lock.RLock()
	defer g.lock.RUnlock()

	return g.nodeKeys[k]
}

func (g *graph) Associate(from Node, kind EdgeKind, to Node, optionalContext ...interface{}) (Edge, error) {
	fromNode := g.nodeKeys[from.NodeKey()]
	if fromNode == nil {
		return nil, ErrNoSuchNode{Node: from, context: "From"}
	}
	toNode := g.nodeKeys[to.NodeKey()]
	if toNode == nil {
		return nil, ErrNoSuchNode{Node: to, context: "To"}
	}

	g.lock.Lock()
	defer g.lock.Unlock()

	// add a new graph builder if this is a new kind
	if _, has := g.directed[kind]; !has {
		g.directed[kind] = newDirected(g)
	}

	dg := g.directed[kind]

	if dg.Node(fromNode.id) == nil {
		dg.AddNode(fromNode)
	}
	if dg.Node(toNode.id) == nil {
		dg.AddNode(toNode)
	}

	new := dg.NewEdge(fromNode, toNode)
	dg.SetEdge(new)

	dg.edges[new] = &edge{
		kind:    kind,
		to:      to,
		from:    from,
		context: optionalContext,
	}

	return dg.edges[new], nil

}

func (g *graph) Edge(from Node, kind EdgeKind, to Node) Edge {
	g.lock.RLock()
	defer g.lock.RUnlock()

	directed, has := g.directed[kind]
	if !has {
		return nil
	}

	args := directed.gonum(from, to)
	if args[0] == nil || args[1] == nil {
		return nil
	}

	return directed.edges[directed.Edge(args[0].ID(), args[1].ID())]
}

type nodesOrEdges struct {
	nodes func() Nodes
	edges func() Edges
}

func (q *nodesOrEdges) Nodes() Nodes {
	return q.nodes()
}

func (q *nodesOrEdges) Edges() Edges {
	return q.edges()
}

func (g *graph) From(from Node, kind EdgeKind) NodesOrEdges {
	return &nodesOrEdges{
		nodes: func() Nodes { return g.find(kind, from, false) },
		edges: func() Edges { return g.findEdges(kind, from, false) },
	}
}

func (g *graph) To(to Node, kind EdgeKind) NodesOrEdges {
	return &nodesOrEdges{
		nodes: func() Nodes { return g.find(kind, to, true) },
		edges: func() Edges { return g.findEdges(kind, to, true) },
	}
}

func (g *graph) find(kind EdgeKind, x Node, to bool) (nodes Nodes) {
	g.lock.RLock()
	defer g.lock.RUnlock()

	ch := make(chan Node)
	nodes = ch

	directed, has := g.directed[kind]
	if !has {
		close(ch)
		return
	}

	arg, has := g.nodeKeys[x.NodeKey()]
	if !has {
		close(ch)
		return
	}

	go func() {
		defer close(ch)

		var result gonum.Nodes
		if to {
			result = directed.To(arg.ID())
		} else {
			result = directed.From(arg.ID())
		}

		for {
			if next := result.Next(); !next {
				break
			}
			ch <- result.Node().(*node).Node
		}
	}()

	return
}

func (g *graph) findEdges(kind EdgeKind, x Node, to bool) (edges Edges) {
	g.lock.RLock()
	defer g.lock.RUnlock()

	ch := make(chan Edge)
	edges = ch

	directed, has := g.directed[kind]
	if !has {
		close(ch)
		return
	}

	arg, has := g.nodeKeys[x.NodeKey()]
	if !has {
		close(ch)
		return
	}

	go func() {
		defer close(ch)

		var result gonum.Nodes
		if to {
			result = directed.To(arg.ID())
		} else {
			result = directed.From(arg.ID())
		}

		for {
			if next := result.Next(); !next {
				break
			}

			if to {
				ch <- directed.edges[directed.Edge(result.Node().ID(), arg.ID())]
			} else {
				ch <- directed.edges[directed.Edge(arg.ID(), result.Node().ID())]
			}
		}
	}()

	return
}
