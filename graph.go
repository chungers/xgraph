package xgraph // import "github.com/orkestr8/xgraph"

import (
	"sync"
)

type node struct {
	Node
	ids map[EdgeKind]int64
}

type edge struct {
	from Node
	to   Node
	kind EdgeKind
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

type graph struct {
	Options
	nodes    map[Node]interface{}
	directed map[EdgeKind]*directed
	nodeKeys map[string]Node

	lock sync.RWMutex
}

func newGraph(options Options) *graph {
	return &graph{
		Options:  options,
		nodes:    map[Node]interface{}{},
		nodeKeys: map[string]Node{},
		directed: map[EdgeKind]*directed{},
	}
}

/*
 Add registers the given Nodes to the graph.  Duplicate key but with different identity is not allowed.
*/
func (g *graph) Add(n Node, other ...Node) error {
	g.lock.Lock()
	defer g.lock.Unlock()

	for _, add := range append([]Node{n}, other...) {
		found, has := g.nodeKeys[string(add.NodeKey())]
		if !has {
			g.nodes[add] = &node{Node: add}
			g.nodeKeys[string(add.NodeKey())] = add
		} else if found != add {
			return ErrDuplicateKey{add}
		}
	}

	return nil
}

func (g *graph) Has(n Node) bool {
	g.lock.RLock()
	defer g.lock.RUnlock()

	_, has := g.nodes[n]
	return has
}

func (g *graph) Node(k NodeKey) Node {
	g.lock.RLock()
	defer g.lock.RUnlock()

	return g.nodeKeys[string(k)]
}

func (g *graph) Associate(from Node, kind EdgeKind, to Node) (Edge, error) {
	// first check for proper node membership
	if !g.Has(from) {
		return nil, ErrNoSuchNode{Node: from, context: "From"}
	}
	if !g.Has(to) {
		return nil, ErrNoSuchNode{Node: to, context: "To"}
	}

	g.lock.Lock()
	defer g.lock.Unlock()

	// add a new graph builder if this is a new kind
	if _, has := g.directed[kind]; !has {
		g.directed[kind] = newDirected()
	}

	dg := g.directed[kind]

	fn := dg.gonum(from)[0]
	if fn == nil {
		fn = dg.add(from)
	}

	tn := dg.gonum(to)[0]
	if tn == nil {
		tn = dg.add(to)
	}

	dg.SetEdge(dg.NewEdge(fn, tn))

	return &edge{
		kind: kind,
		to:   to,
		from: from,
	}, nil

}

func (g *graph) Edge(from Node, kind EdgeKind, to Node) bool {
	g.lock.RLock()
	defer g.lock.RUnlock()

	directed, has := g.directed[kind]
	if !has {
		return false
	}

	args := directed.gonum(from, to)
	if args[0] == nil || args[1] == nil {
		return false
	}
	return directed.HasEdgeBetween(args[0].ID(), args[1].ID())
}
