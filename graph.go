package xgraph // import "github.com/orkestr8/xgraph"

import (
	"sync"

	gonum "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
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
func (e *edge) Reverse() Edge {
	return &edge{
		kind: e.kind,
		from: e.to,
		to:   e.from,
	}
}

type graph struct {
	Options
	nodes    map[Node]*node
	ids      map[EdgeKind]map[int64]*node
	builders map[EdgeKind]gonum.DirectedBuilder
	lookup   map[EdgeKind]map[int64]*node

	lock sync.RWMutex
}

func newGraph(options Options) *graph {
	return &graph{
		Options:  options,
		nodes:    map[Node]*node{},
		ids:      map[EdgeKind]map[int64]*node{},
		builders: map[EdgeKind]gonum.DirectedBuilder{},
	}
}

func (g *graph) Add(n Node, other ...Node) error {
	g.lock.Lock()
	defer g.lock.Unlock()

	for _, add := range append([]Node{n}, other...) {
		found, has := g.nodes[add]
		if has && found.Node != add {
			return ErrDuplicateKey{n}
		}
	}

	for _, add := range append([]Node{n}, other...) {
		g.nodes[add] = &node{Node: add}
	}

	return nil
}

func (g *graph) Has(n Node) bool {
	g.lock.RLock()
	defer g.lock.RUnlock()

	_, has := g.nodes[n]
	return has
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
	if _, has := g.builders[kind]; !has {
		g.builders[kind] = simple.NewDirectedGraph() // TODO: copy graph, mutate, then commit at the end?
	}
	// mapping of node id to Node
	if _, has := g.ids[kind]; !has {
		g.ids[kind] = map[int64]*node{}
	}

	// get the node id for the Node to add edges
	if g.nodes[from].ids == nil {
		g.nodes[from].ids = map[EdgeKind]int64{}
	}
	if g.nodes[to].ids == nil {
		g.nodes[to].ids = map[EdgeKind]int64{}
	}

	get_id := func(b gonum.DirectedBuilder, _k EdgeKind, _n Node) (_id int64) {
		if _, has := g.nodes[_n].ids[_k]; !has {
			// add the node to the graph
			new := g.builders[_k].NewNode()
			g.builders[_k].AddNode(new)
			id := new.ID()
			g.ids[_k][id] = g.nodes[_n]
			g.nodes[_n].ids[_k] = id
		}
		return g.nodes[_n].ids[_k]
	}

	builder := g.builders[kind]

	fromID := get_id(builder, kind, from)
	toID := get_id(builder, kind, to)

	new := builder.NewEdge(builder.Node(fromID), builder.Node(toID))
	builder.SetEdge(new)

	return &edge{
		kind: kind,
		to:   to,
		from: from,
	}, nil

}

func (g *graph) Edge(from Node, kind EdgeKind, to Node) bool {
	g.lock.RLock()
	defer g.lock.RUnlock()

	builder, has := g.builders[kind]
	if !has {
		return false
	}

	_from, has := g.nodes[from]
	if !has {
		return false
	}

	_to, has := g.nodes[to]
	if !has {
		return false
	}

	return builder.HasEdgeBetween(_from.ids[kind], _to.ids[kind])
}
